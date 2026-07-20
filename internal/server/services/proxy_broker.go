package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"sync"
	"time"

	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

const pendingWorkTTL = 15 * time.Second

type ProxyBroker struct {
	Logger      *zap.Logger
	pendingMu   sync.Mutex
	pendingWork map[string]*pendingWorkItem
}

type pendingWorkItem struct {
	userConn net.Conn
	certFP   [sha256.Size]byte
}

func (s *ProxyBroker) StartWorkConn(workTLS net.Conn, payload []byte) error {
	var msg sharedprotocol.StartWorkConnMsg
	if err := sharedprotocol.Decode(payload, &msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse StartWorkConnMsg: %v", err))
		return fmt.Errorf("failed to parse StartWorkConnMsg")
	}
	if msg.WorkID == "" {
		_ = workTLS.Close()
		return fmt.Errorf("work_id is empty")
	}
	userConn, ok := s.takePendingIfCertMatches(msg.WorkID, workTLS)
	if !ok {
		s.Logger.Warn(fmt.Sprintf("No matching pending work or client certificate mismatch for work_id=%s", msg.WorkID))
		_ = workTLS.Close()
		return fmt.Errorf("unknown or expired work_id, or certificate mismatch")
	}
	go network.Pipe(userConn, workTLS)
	return nil
}

func (s *ProxyBroker) Register(workID string, userConn net.Conn, certFP [sha256.Size]byte) {
	s.pendingMu.Lock()
	if s.pendingWork == nil {
		s.pendingWork = make(map[string]*pendingWorkItem)
	}
	s.pendingWork[workID] = &pendingWorkItem{
		userConn: userConn,
		certFP:   certFP,
	}
	s.pendingMu.Unlock()

	time.AfterFunc(pendingWorkTTL, func() {
		s.pendingMu.Lock()
		p, ok := s.pendingWork[workID]
		if ok {
			delete(s.pendingWork, workID)
		}
		s.pendingMu.Unlock()
		if ok {
			_ = p.userConn.Close()
			s.Logger.Warn(fmt.Sprintf("Timed out waiting for work channel; closed user connection: workID=%s", workID))
		}
	})
}

func (s *ProxyBroker) Remove(workID string) net.Conn {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingWork == nil {
		return nil
	}
	p, ok := s.pendingWork[workID]
	if !ok {
		return nil
	}
	delete(s.pendingWork, workID)
	return p.userConn
}

func (s *ProxyBroker) takePendingIfCertMatches(workID string, workTLS net.Conn) (net.Conn, bool) {
	workFP, err := sharedcerts.ClientLeafCertSHA256(workTLS)
	if err != nil {
		s.Logger.Warn(fmt.Sprintf("StartWorkConn: read work TLS client cert: %v", err))
		return nil, false
	}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingWork == nil {
		return nil, false
	}
	p, ok := s.pendingWork[workID]
	if !ok {
		return nil, false
	}
	if subtle.ConstantTimeCompare(p.certFP[:], workFP[:]) != 1 {
		s.Logger.Warn(fmt.Sprintf("StartWorkConn rejected: client certificate does not match control channel (work_id=%s)", workID))
		return nil, false
	}
	delete(s.pendingWork, workID)
	return p.userConn, true
}
