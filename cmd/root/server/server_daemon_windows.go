//go:build windows

package server

import "fmt"

func runServerDaemonStart(configFile string, pidPath string) error {
	return fmt.Errorf("server --daemon is not supported on Windows")
}

func runServerDaemonStop(pidPath string) error {
	return fmt.Errorf("server --daemon is not supported on Windows")
}

func runServerDaemonReload(pidPath string) error {
	return fmt.Errorf("server --daemon is not supported on Windows")
}

func removeServerPidFileIfSelf(pidPath string) {}
