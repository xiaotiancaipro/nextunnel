package utils

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const DaemonReadyEnv = "NEXTUNNEL_DAEMON_READY_FD"

func EnsureStalePidFileCleared(file string) error {
	pid, err := ReadPidFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		_ = os.Remove(file)
		return nil
	}
	if ProcessAlive(pid) {
		return fmt.Errorf("already running (pid %d)", pid)
	}
	return os.Remove(file)
}

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func ReadPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0, fmt.Errorf("empty pid file")
	}
	return strconv.Atoi(s)
}

func WritePidFile(path string, pid int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0o600)
}

func NotifyDaemonReady() {
	NotifyDaemonStartStatus("ready")
}

func NotifyDaemonStartFailure(err error) {
	if err == nil {
		return
	}
	NotifyDaemonStartStatus("error: " + err.Error())
}

func NotifyDaemonStartStatus(status string) {
	fdStr := strings.TrimSpace(os.Getenv(DaemonReadyEnv))
	if fdStr == "" || status == "" {
		return
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil || fd < 0 {
		return
	}
	pipe := os.NewFile(uintptr(fd), DaemonReadyEnv)
	if pipe == nil {
		return
	}
	_, _ = pipe.WriteString(status + "\n")
	_ = pipe.Close()
	_ = os.Unsetenv(DaemonReadyEnv)
}

func AwaitDaemonReady(readyR *os.File) error {
	type daemonReadyResult struct {
		status string
		err    error
	}

	resultCh := make(chan daemonReadyResult, 1)
	go func() {
		data, err := io.ReadAll(readyR)
		resultCh <- daemonReadyResult{
			status: strings.TrimSpace(string(data)),
			err:    err,
		}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			return fmt.Errorf("read readiness status: %w", result.err)
		}
		switch {
		case result.status == "ready":
			return nil
		case result.status == "":
			return fmt.Errorf("process exited before reporting ready")
		case strings.HasPrefix(result.status, "error: "):
			return fmt.Errorf("process exited before reporting ready: %w", fmt.Errorf(strings.TrimSpace(strings.TrimPrefix(result.status, "error: "))))
		default:
			return fmt.Errorf("unexpected readiness status %q", result.status)
		}
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timed out waiting for daemon readiness")
	}
}
