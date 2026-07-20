package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
)

type Server struct {
	Config *configs.Server
}

func (s *Server) Dial(c *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addr := s.AddrStr()
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, c)
	if err != nil {
		return nil, fmt.Errorf("failed to dial server %s: %w", addr, err)
	}
	return conn, nil
}

func (s *Server) AddrStr() string {
	return net.JoinHostPort(s.Config.Host, strconv.Itoa(s.Config.Port))
}
