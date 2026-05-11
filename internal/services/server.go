package services

import (
	"net"
	"strconv"

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

func (s *Server) AddrStr() string {
	return net.JoinHostPort(s.config.Addr, strconv.Itoa(s.config.Port))
}
