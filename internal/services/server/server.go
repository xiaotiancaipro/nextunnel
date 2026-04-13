package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

type Server struct {
	logger          *logrus.Logger
	runtimeMu       sync.RWMutex
	runtime         *runtimeConfig
	listenerMu      sync.Mutex
	currentListener *listenerState
	legacyListeners []*listenerState
	mu              sync.RWMutex
	clients         map[string]*ControlSession
	activeClients   map[string]string
	proxies         map[string]*proxyEntry
	pendingWork     map[string]chan net.Conn
	pendingWorkMu   sync.Mutex
	stopCh          chan struct{}
	stopOnce        sync.Once
}

type runtimeConfig struct {
	bindPort  int
	token     string
	tls       configs.ServerTLSConfigs
	tlsConfig *tls.Config
	ipFilter  *IpFilter
}

type listenerState struct {
	listener net.Listener
	runtime  *runtimeConfig
}

type Params struct {
	BindPort int
	Token    string
	TLS      configs.ServerTLSConfigs
	IPFilter configs.ServerIPFilterConfigs
	Logger   *logrus.Logger
}

type proxyEntry struct {
	name       string
	remotePort int
	clientID   string
	ownerRunID string
	listener   net.Listener
}

func NewServer(params *Params) (*Server, error) {
	if params.BindPort <= 0 || params.BindPort > 65535 {
		return nil, fmt.Errorf("invalid bind port: %d", params.BindPort)
	}
	rt, err := buildRuntimeConfig(&configs.ServerConfigs{
		BindPort: params.BindPort,
		Token:    params.Token,
		TLS:      params.TLS,
		IPFilter: params.IPFilter,
	})
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:        params.Logger,
		runtime:       rt,
		clients:       make(map[string]*ControlSession),
		activeClients: make(map[string]string),
		proxies:       make(map[string]*proxyEntry),
		pendingWork:   make(map[string]chan net.Conn),
		stopCh:        make(chan struct{}),
	}, nil
}

func buildRuntimeConfig(cfg *configs.ServerConfigs) (*runtimeConfig, error) {

	allow, err := utils.NormalizeIPList(cfg.IPFilter.Allow)
	if err != nil {
		return nil, fmt.Errorf("invalid allow ip list: %w", err)
	}

	deny, err := utils.NormalizeIPList(cfg.IPFilter.Deny)
	if err != nil {
		return nil, fmt.Errorf("invalid deny ip list: %w", err)
	}

	var tlsConfig *tls.Config
	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
			return nil, fmt.Errorf("tls cert_file and key_file are required when tls is enabled")
		}
		if cfg.TLS.CAFile == "" {
			return nil, fmt.Errorf("tls ca_file is required when tls is enabled")
		}
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load tls certificate: %w", err)
		}
		caCert, err := os.ReadFile(cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read tls ca_file: %w", err)
		}
		clientCAs := x509.NewCertPool()
		if ok := clientCAs.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to append tls ca_file to client cert pool")
		}
		tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    clientCAs,
		}
	}

	return &runtimeConfig{
		bindPort:  cfg.BindPort,
		token:     cfg.Token,
		tls:       cfg.TLS,
		tlsConfig: tlsConfig,
		ipFilter: &IpFilter{
			Allow: allow,
			Deny:  deny,
		},
	}, nil

}

func (s *Server) snapshotRuntime() *runtimeConfig {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.runtime
}

func (s *Server) Start() error {
	rt := s.snapshotRuntime()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", rt.bindPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", rt.bindPort, err)
	}
	state := &listenerState{listener: ln, runtime: rt}
	s.listenerMu.Lock()
	s.currentListener = state
	s.listenerMu.Unlock()
	go s.acceptLoop(state)
	return nil
}

func (s *Server) ApplyConfig(cfg *configs.ServerConfigs) error {

	nextRuntime, err := buildRuntimeConfig(cfg)
	if err != nil {
		return err
	}

	currentRuntime := s.snapshotRuntime()
	var nextListener *listenerState
	if nextRuntime.bindPort != currentRuntime.bindPort {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", nextRuntime.bindPort))
		if err != nil {
			return fmt.Errorf("failed to listen on port %d: %w", nextRuntime.bindPort, err)
		}
		nextListener = &listenerState{listener: ln, runtime: nextRuntime}
	}

	s.runtimeMu.Lock()
	s.runtime = nextRuntime
	s.runtimeMu.Unlock()

	if nextListener != nil {
		s.listenerMu.Lock()
		if s.currentListener != nil {
			s.legacyListeners = append(s.legacyListeners, s.currentListener)
		}
		s.currentListener = nextListener
		s.listenerMu.Unlock()

		go s.acceptLoop(nextListener)
		s.logger.Infof("server bind port reloaded: %d -> %d", currentRuntime.bindPort, nextRuntime.bindPort)
	} else {
		s.listenerMu.Lock()
		if s.currentListener != nil {
			s.currentListener.runtime = nextRuntime
		}
		s.listenerMu.Unlock()
	}

	s.logger.Infof(
		"server runtime config reloaded (bind_port=%d, tls=%t, ip_filter_allow=%d, ip_filter_deny=%d)",
		nextRuntime.bindPort,
		nextRuntime.tls.Enabled,
		len(nextRuntime.ipFilter.Allow),
		len(nextRuntime.ipFilter.Deny),
	)
	return nil

}

func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.listenerMu.Lock()
		if s.currentListener != nil {
			_ = s.currentListener.listener.Close()
		}
		for _, legacy := range s.legacyListeners {
			if legacy != nil && legacy.listener != nil {
				_ = legacy.listener.Close()
			}
		}
		s.listenerMu.Unlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, proxy := range s.proxies {
			if proxy.listener != nil {
				_ = proxy.listener.Close()
			}
		}
		for _, sess := range s.clients {
			_ = sess.conn.Close()
		}
	})
}

func (s *Server) acceptLoop(state *listenerState) {
	for {
		rawConn, err := state.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
			}
			s.logger.Errorf("Accept failed: %v", err)
			return
		}
		s.listenerMu.Lock()
		runtime := state.runtime
		s.listenerMu.Unlock()
		go s.handleAcceptedConn(rawConn, runtime)
	}
}

func (s *Server) handleAcceptedConn(rawConn net.Conn, runtime *runtimeConfig) {
	conn, err := s.wrapIncomingConn(rawConn, runtime)
	if err != nil {
		s.logger.Errorf("Failed to initialize incoming connection [%s]: %v", rawConn.RemoteAddr(), err)
		_ = rawConn.Close()
		return
	}
	s.handleIncoming(conn)
}

func (s *Server) wrapIncomingConn(rawConn net.Conn, runtime *runtimeConfig) (net.Conn, error) {
	if !runtime.tls.Enabled {
		return rawConn, nil
	}
	tlsConn := tls.Server(rawConn, runtime.tlsConfig)
	_ = tlsConn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.SetDeadline(time.Time{})
		return nil, fmt.Errorf("tls handshake failed: %w", err)
	}
	_ = tlsConn.SetDeadline(time.Time{})
	return tlsConn, nil
}

func (s *Server) handleIncoming(conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	if err != nil {
		s.logger.Errorf("Failed to read first message [%s]: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})
	switch msgType {
	case utils.MsgLogin:
		s.handleControlConn(conn, payload)
	case utils.MsgStartWorkConn:
		s.handleWorkConn(conn, payload)
	default:
		s.logger.Warnf("Unknown first message type 0x%02x [%s]", msgType, conn.RemoteAddr())
		_ = conn.Close()
	}
}

func (s *Server) handleControlConn(conn net.Conn, payload []byte) {

	var loginMsg utils.LoginMsg
	if err := utils.Decode(payload, &loginMsg); err != nil {
		s.logger.Errorf("Failed to parse LoginMsg: %v", err)
		_ = conn.Close()
		return
	}

	if loginMsg.ClientID == "" {
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "client_id cannot be empty"})
		_ = conn.Close()
		return
	}

	rt := s.snapshotRuntime()
	if loginMsg.Token != rt.token {
		s.logger.Warnf("Authentication failed [%s]: token mismatch", conn.RemoteAddr())
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "authentication failed"})
		_ = conn.Close()
		return
	}

	sess := &ControlSession{
		runID:    uuid.New().String(),
		clientID: loginMsg.ClientID,
		conn:     conn,
		stopCh:   make(chan struct{}),
	}

	s.mu.Lock()
	s.clients[sess.runID] = sess
	s.mu.Unlock()

	s.logger.Infof("Client connected [%s], clientID=%s, runID=%s", conn.RemoteAddr(), sess.clientID, sess.runID)
	defer s.removeClient(sess.runID)

	if err := sess.WriteMsg(utils.MsgLoginResp, utils.LoginRespMsg{RunID: sess.runID}); err != nil {
		s.logger.Errorf("Failed to send LoginResp: %v", err)
		return
	}

	for {
		msgType, payload, err := utils.ReadMsg(sess.conn)
		if err != nil {
			s.logger.Infof("Client control connection disconnected clientID=%s runID=%s: %v", sess.clientID, sess.runID, err)
			return
		}

		switch msgType {
		case utils.MsgApplyConfig:
			s.handleApplyConfig(sess, payload)
		case utils.MsgPing:
			if err := sess.WriteMsg(utils.MsgPong, utils.PongMsg{}); err != nil {
				s.logger.Errorf("Failed to send Pong: %v", err)
				return
			}
		default:
			s.logger.Warnf("Unknown message received on control connection 0x%02x runID=%s", msgType, sess.runID)
		}
	}

}

func (s *Server) handleApplyConfig(sess *ControlSession, payload []byte) {
	var msg utils.ApplyConfigMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.sendApplyConfigResp(sess, fmt.Sprintf("failed to parse ApplyConfigMsg: %v", err))
		return
	}
	if err := s.applyClientConfig(sess, msg.Proxies); err != nil {
		s.sendApplyConfigResp(sess, err.Error())
		return
	}
	s.sendApplyConfigResp(sess, "")
}

func (s *Server) sendApplyConfigResp(sess *ControlSession, errMsg string) {
	if err := sess.WriteMsg(utils.MsgApplyConfigResp, utils.ApplyConfigRespMsg{Error: errMsg}); err != nil {
		s.logger.Errorf("Failed to send ApplyConfigResp runID=%s: %v", sess.runID, err)
	}
}

func (s *Server) applyClientConfig(sess *ControlSession, proxies []utils.ApplyConfigProxyMsg) error {

	desired := make(map[string]utils.ApplyConfigProxyMsg, len(proxies))
	for i, proxy := range proxies {
		if proxy.Name == "" {
			return fmt.Errorf("proxies[%d].name cannot be empty", i)
		}
		if proxy.Type != "tcp" {
			return fmt.Errorf("proxies[%d].type must be tcp", i)
		}
		if proxy.RemotePort <= 0 || proxy.RemotePort > 65535 {
			return fmt.Errorf("invalid proxies[%d].remote_port: %d", i, proxy.RemotePort)
		}
		if _, exists := desired[proxy.Name]; exists {
			return fmt.Errorf("duplicate proxy name: %s", proxy.Name)
		}
		desired[proxy.Name] = proxy
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, proxy := range proxies {
		if existing, ok := s.proxies[proxy.Name]; ok && existing.clientID != sess.clientID {
			return fmt.Errorf("proxy name already exists: %s", proxy.Name)
		}
	}

	opened := make(map[string]net.Listener)
	closeOpened := func() {
		for _, ln := range opened {
			_ = ln.Close()
		}
	}
	for _, proxy := range proxies {
		existing, ok := s.proxies[proxy.Name]
		if ok && existing.clientID == sess.clientID && existing.remotePort == proxy.RemotePort {
			continue
		}

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", proxy.RemotePort))
		if err != nil {
			closeOpened()
			return fmt.Errorf("failed to listen on port %d for proxy %s: %w", proxy.RemotePort, proxy.Name, err)
		}
		opened[proxy.Name] = ln
	}

	oldActiveRunID := s.activeClients[sess.clientID]
	var started []*proxyEntry
	var toClose []net.Listener

	for _, proxy := range proxies {
		existing, ok := s.proxies[proxy.Name]
		if ok && existing.clientID == sess.clientID && existing.remotePort == proxy.RemotePort {
			existing.ownerRunID = sess.runID
			continue
		}

		nextEntry := &proxyEntry{
			name:       proxy.Name,
			remotePort: proxy.RemotePort,
			clientID:   sess.clientID,
			ownerRunID: sess.runID,
			listener:   opened[proxy.Name],
		}
		if ok && existing.clientID == sess.clientID {
			toClose = append(toClose, existing.listener)
		}
		s.proxies[proxy.Name] = nextEntry
		started = append(started, nextEntry)
	}

	for name, existing := range s.proxies {
		if existing.clientID != sess.clientID {
			continue
		}
		if _, keep := desired[name]; keep {
			continue
		}
		if oldActiveRunID != "" && existing.ownerRunID != oldActiveRunID {
			continue
		}
		delete(s.proxies, name)
		toClose = append(toClose, existing.listener)
	}

	s.activeClients[sess.clientID] = sess.runID

	for _, entry := range started {
		go s.proxyAcceptLoop(entry)
	}
	for _, ln := range toClose {
		_ = ln.Close()
	}

	s.logger.Infof("Client config applied: clientID=%s, runID=%s, proxies=%d", sess.clientID, sess.runID, len(proxies))
	return nil

}

func (s *Server) proxyAcceptLoop(entry *proxyEntry) {

	defer func() {
		_ = entry.listener.Close()
		s.mu.Lock()
		if current, ok := s.proxies[entry.name]; ok && current == entry {
			delete(s.proxies, entry.name)
		}
		s.mu.Unlock()
		s.logger.Infof("Proxy stopped: name=%s, port=%d", entry.name, entry.remotePort)
	}()

	for {

		userConn, err := entry.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Infof("Proxy [%s] accept loop exiting: %v", entry.name, err)
				return
			}
		}

		rt := s.snapshotRuntime()
		allowed, srcIP, reason := rt.ipFilter.AllowIP(userConn.RemoteAddr())
		if !allowed {
			s.logger.Warnf("User connection rejected by ip filter: proxy=%s, src=%s, ip=%s, reason=%s", entry.name, userConn.RemoteAddr(), srcIP, reason)
			_ = userConn.Close()
			continue
		}

		s.mu.RLock()
		ownerRunID := entry.ownerRunID
		sess := s.clients[ownerRunID]
		s.mu.RUnlock()
		if sess == nil {
			s.logger.Warnf("No active owner session for proxy=%s runID=%s", entry.name, ownerRunID)
			_ = userConn.Close()
			continue
		}

		s.logger.Infof("User connection arrived: proxy=%s, src=%s, ip=%s, runID=%s", entry.name, userConn.RemoteAddr(), srcIP, ownerRunID)

		go s.bridgeUserConn(userConn, entry.name, sess)

	}

}

func (s *Server) bridgeUserConn(userConn net.Conn, proxyName string, sess *ControlSession) {

	defer func() { _ = userConn.Close() }()

	workID := uuid.New().String()
	workCh := make(chan net.Conn, 1)

	s.pendingWorkMu.Lock()
	s.pendingWork[workID] = workCh
	s.pendingWorkMu.Unlock()

	defer func() {
		s.pendingWorkMu.Lock()
		delete(s.pendingWork, workID)
		s.pendingWorkMu.Unlock()
	}()

	msg := utils.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: proxyName,
	}
	if err := sess.WriteMsg(utils.MsgNewWorkConn, msg); err != nil {
		s.logger.Errorf("Failed to send NewWorkConn: %v", err)
		return
	}

	select {
	case workConn := <-workCh:
		s.logger.Debugf("Work connection ready: workID=%s, proxy=%s", workID, proxyName)
		utils.Pipe(userConn, workConn)
	case <-time.After(10 * time.Second):
		s.logger.Warnf("Timed out waiting for work connection: workID=%s, proxy=%s", workID, proxyName)
	}

}

func (s *Server) handleWorkConn(conn net.Conn, payload []byte) {

	var msg utils.StartWorkConnMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.logger.Errorf("Failed to parse StartWorkConnMsg: %v", err)
		_ = conn.Close()
		return
	}

	s.pendingWorkMu.Lock()
	ch, ok := s.pendingWork[msg.WorkID]
	s.pendingWorkMu.Unlock()

	if !ok {
		s.logger.Warnf("Received unknown work connection workID=%s", msg.WorkID)
		_ = conn.Close()
		return
	}

	ch <- conn

}

func (s *Server) removeClient(runID string) {

	var toClose []net.Listener

	s.mu.Lock()
	sess, ok := s.clients[runID]
	if ok {
		close(sess.stopCh)
		_ = sess.conn.Close()
		delete(s.clients, runID)
		if s.activeClients[sess.clientID] == runID {
			delete(s.activeClients, sess.clientID)
		}
	}
	for name, proxy := range s.proxies {
		if proxy.ownerRunID != runID {
			continue
		}
		delete(s.proxies, name)
		toClose = append(toClose, proxy.listener)
		s.logger.Infof("Proxy removed: name=%s (client disconnected)", name)
	}
	s.mu.Unlock()

	for _, ln := range toClose {
		_ = ln.Close()
	}

	s.logger.Infof("Client session cleaned up: runID=%s", runID)

}
