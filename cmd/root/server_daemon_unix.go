//go:build !windows

package root

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func ensureStalePidFileCleared(path string) error {
	pid, err := utils.ReadPidFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		_ = os.Remove(path)
		return nil
	}
	if processAlive(pid) {
		return fmt.Errorf("server already running (pid %d)", pid)
	}
	return os.Remove(path)
}

func runServerDaemonStart(configFile string, pidPath string) error {
	if err := ensureStalePidFileCleared(pidPath); err != nil {
		return err
	}
	absConfig, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	absExe, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	logPath := utils.LogPathBesideConfig(absConfig)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	cmd := exec.Command(absExe, "server", "--config", absConfig)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", serverDaemonPidEnvKey, pidPath))
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start daemon process: %w", err)
	}
	_ = logFile.Close()
	pid := cmd.Process.Pid
	if err := utils.WritePidFile(pidPath, pid); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return fmt.Errorf("write pid file: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release process: %w", err)
	}
	return nil
}

func runServerDaemonStop(pidPath string) error {
	pid, err := utils.ReadPidFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !processAlive(pid) {
		_ = os.Remove(pidPath)
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal TERM to pid %d: %w", pid, err)
	}
	return nil
}

func runServerDaemonReload(pidPath string) error {
	pid, err := utils.ReadPidFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pid file not found: %s", pidPath)
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !processAlive(pid) {
		return fmt.Errorf("no process with pid %d", pid)
	}
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("signal HUP to pid %d: %w", pid, err)
	}
	return nil
}

func removeServerPidFileIfSelf(pidPath string) {
	if pidPath == "" {
		return
	}
	pid, err := utils.ReadPidFile(pidPath)
	if err != nil {
		return
	}
	if pid == os.Getpid() {
		_ = os.Remove(pidPath)
	}
}
