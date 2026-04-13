package server

import (
	"net"
	"sync"

	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

type ControlSession struct {
	runID  string
	conn   net.Conn
	mu     sync.Mutex
	stopCh chan struct{}
}

func (s *ControlSession) WriteMsg(msgType byte, payload interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return utils.WriteMsg(s.conn, msgType, payload)
}
