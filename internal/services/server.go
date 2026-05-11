package services

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"go.uber.org/zap"
)

type Server struct {
	config *configs.Server
	logger *zap.Logger
}

func NewServer(config *configs.Configs, logger *zap.Logger) *Server {
	return &Server{
		config: config.Server,
		logger: logger,
	}
}

func (s *Server) DialServer(c *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addr := s.AddrStr()
	return tls.DialWithDialer(dialer, "tcp", addr, c)
}

func (s *Server) AddrStr() string {
	return net.JoinHostPort(s.config.Addr, strconv.Itoa(s.config.Port))
}
