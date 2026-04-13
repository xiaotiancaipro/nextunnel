package server

import (
	"net"

	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

type IpFilter struct {
	Allow map[string]struct{}
	Deny  map[string]struct{}
}

func (f *IpFilter) AllowIP(remoteAddr net.Addr) (bool, string, string) {

	if len(f.Allow) == 0 && len(f.Deny) == 0 {
		return true, "", ""
	}

	host := remoteAddr.String()
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	ip, err := utils.NormalizeIP(host)
	if err != nil {
		return false, "", "failed to parse remote ip"
	}
	if _, blocked := f.Deny[ip]; blocked {
		return false, ip, "matched deny list"
	}
	if len(f.Allow) > 0 {
		if _, ok := f.Allow[ip]; !ok {
			return false, ip, "not in allow list"
		}
	}
	return true, ip, ""

}
