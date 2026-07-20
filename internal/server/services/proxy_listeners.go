package services

import (
	"fmt"
	"net"
	"sync"

	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
)

type ProxyListeners struct {
	mu    sync.Mutex
	items map[string]proxyListenerItem
}

type proxyListenerItem struct {
	ln         net.Listener
	remotePort int
}

func (s *ProxyListeners) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, item := range s.items {
		_ = item.ln.Close()
		delete(s.items, name)
	}
}

func (s *ProxyListeners) reconcile(desired map[string]sharedprotocol.ProxiesApplyMsgItem) (map[string]net.Listener, error) {

	s.mu.Lock()
	if s.items == nil {
		s.items = make(map[string]proxyListenerItem)
	}
	stale := make([]net.Listener, 0)
	for name, item := range s.items {
		want, ok := desired[name]
		if !ok || want.RemotePort != item.remotePort {
			stale = append(stale, item.ln)
			delete(s.items, name)
		}
	}
	s.mu.Unlock()

	for _, ln := range stale {
		_ = ln.Close()
	}

	opened := make(map[string]net.Listener)
	rollback := func() {
		for _, ln := range opened {
			_ = ln.Close()
		}
		s.mu.Lock()
		for name := range opened {
			delete(s.items, name)
		}
		s.mu.Unlock()
	}

	for name, proxy := range desired {
		s.mu.Lock()
		_, exists := s.items[name]
		s.mu.Unlock()
		if exists {
			continue
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", proxy.RemotePort))
		if err != nil {
			rollback()
			return nil, fmt.Errorf("failed to listen on port %d: %w", proxy.RemotePort, err)
		}
		opened[name] = ln
		s.mu.Lock()
		if s.items == nil {
			s.items = make(map[string]proxyListenerItem)
		}
		s.items[name] = proxyListenerItem{ln: ln, remotePort: proxy.RemotePort}
		s.mu.Unlock()
	}
	return opened, nil

}
