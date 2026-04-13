package utils

import (
	"fmt"
	"io"
	"net"
	"strings"
)

func LocalIP(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	return ip
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

func NormalizeIPList(rawIPs []string) (map[string]struct{}, error) {
	if len(rawIPs) == 0 {
		return map[string]struct{}{}, nil
	}
	ips := make(map[string]struct{}, len(rawIPs))
	for _, raw := range rawIPs {
		ip, err := NormalizeIP(raw)
		if err != nil {
			return nil, err
		}
		ips[ip] = struct{}{}
	}
	return ips, nil
}

func NormalizeIP(raw string) (string, error) {
	ipStr := strings.TrimSpace(raw)
	if zoneIdx := strings.Index(ipStr, "%"); zoneIdx >= 0 {
		ipStr = ipStr[:zoneIdx]
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid ip: %s", raw)
	}
	return ip.String(), nil
}
