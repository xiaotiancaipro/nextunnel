package services

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"go.uber.org/zap"
)

type Server struct {
	Config *configs.Server
	Logger *zap.Logger
}

func (s *Server) DialServer(c *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addr := s.AddrStr()
	return tls.DialWithDialer(dialer, "tcp", addr, c)
}

func (s *Server) AddrStr() string {
	return net.JoinHostPort(s.Config.Host, strconv.Itoa(s.Config.Port))
}
