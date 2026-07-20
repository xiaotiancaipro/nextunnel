package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"go.uber.org/zap"
)

type Listener struct {
	Config *configs.Server
	Logger *zap.Logger
}

func (s *Listener) Listen() (net.Listener, error) {
	addr := fmt.Sprintf(":%d", s.Config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to listen on %s: %v", addr, err))
		return nil, fmt.Errorf("failed to listen")
	}
	return listener, nil
}

func (s *Listener) Establish(connRaw net.Conn, tlsConfig *tls.Config) (net.Conn, error) {
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
