package network

import (
	"fmt"
	"io"
	"net"
	"strings"
)

const UnknownIP = "UNKNOWN_IP"

func NormalizeIP(raw string) (*string, error) {
	ipStr := strings.TrimSpace(raw)
	if zoneIdx := strings.Index(ipStr, "%"); zoneIdx >= 0 {
		ipStr = ipStr[:zoneIdx]
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip: %s", raw)
	}
	return new(ip.String()), nil
}

func IsLocalIP(raw string) bool {
	ipStr := strings.TrimSpace(raw)
	if zoneIdx := strings.Index(ipStr, "%"); zoneIdx >= 0 {
		ipStr = ipStr[:zoneIdx]
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

func Pipe(a, b net.Conn) {
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	done := make(chan struct{}, 2)
	copyFn := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyFn(a, b)
	go copyFn(b, a)
	<-done
}
