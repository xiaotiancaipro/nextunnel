package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

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
		return fmt.Errorf("server already running (pid %d)", pid)
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

func ResolvePidPath(configFile, flagPid string) string {
	if flagPid != "" {
		return flagPid
	}
	return configFile + ".pid"
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

func LogPathBesideConfig(configFile string) string {
	if abs, err := filepath.Abs(configFile); err == nil {
		return logPathWithBase(abs)
	}
	return logPathWithBase(configFile)
}

func logPathWithBase(cfgPath string) string {
	base := filepath.Base(cfgPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(filepath.Dir(cfgPath), name+".log")
}
