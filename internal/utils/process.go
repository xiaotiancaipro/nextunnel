package utils

import (
	"os"
	"strconv"
	"strings"
)

const DaemonReadyEnv = "NEXTUNNEL_CLIENT_DAEMON_READY_FD"

func ProcessNotifyDaemonStartFailure(err error) {
	if err == nil {
		return
	}
	ProcessNotifyDaemonStartStatus("error: " + err.Error())
}

func ProcessNotifyDaemonStartStatus(status string) {
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
