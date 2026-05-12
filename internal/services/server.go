package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"go.uber.org/zap"
)

type Server struct {
	Config *configs.Server
	Logger *zap.Logger
}

func (s *Server) Listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", s.AddrStr())
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to listen on %s: %v", s.AddrStr(), err))
		return nil, fmt.Errorf("failed to listen")
	}
	return listener, nil
}

func (s *Server) AddrStr() string {
	return net.JoinHostPort(s.Config.Addr, strconv.Itoa(s.Config.Port))
}

func (s *Server) EstablishConn(connRaw net.Conn, tlsConfig *tls.Config) (net.Conn, error) {
	conn := tls.Server(connRaw, tlsConfig)
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := conn.Handshake(); err != nil {
		_ = conn.SetDeadline(time.Time{})
		s.Logger.Error(fmt.Sprintf("Failed to handshake with TLS connection: %v", err))
		return nil, fmt.Errorf("tls handshake failed")
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}
