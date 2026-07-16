package network

import (
	"fmt"
	"net"
	"strings"
)

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
