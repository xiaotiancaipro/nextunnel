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
	bindPort      int
	token         string
	tls           configs.ServerTLSConfigs
	ipFilter      *IpFilter
	logger        *logrus.Logger
	listener      net.Listener
	mu            sync.RWMutex
	clients       map[string]*ControlSession // runID → control session
	proxies       map[string]*proxyEntry     // proxyName → proxy entry
	pendingWork   map[string]chan net.Conn
	pendingWorkMu sync.Mutex
	stopCh        chan struct{}
	stopOnce      sync.Once
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
	runID      string       // owning client runID
	listener   net.Listener // server listener on remotePort
}

func NewServer(params *Params) (*Server, error) {
	if params.BindPort <= 0 || params.BindPort > 65535 {
		return nil, fmt.Errorf("invalid bind port: %d", params.BindPort)
	}
	allow, err := utils.NormalizeIPList(params.IPFilter.Allow)
	if err != nil {
		return nil, fmt.Errorf("invalid allow ip list: %w", err)
	}
	deny, err := utils.NormalizeIPList(params.IPFilter.Deny)
	if err != nil {
		return nil, fmt.Errorf("invalid deny ip list: %w", err)
	}
	filter := &IpFilter{
		Allow: allow,
		Deny:  deny,
	}
	return &Server{
		bindPort:    params.BindPort,
		token:       params.Token,
		tls:         params.TLS,
		ipFilter:    filter,
		logger:      params.Logger,
		clients:     make(map[string]*ControlSession),
		proxies:     make(map[string]*proxyEntry),
		pendingWork: make(map[string]chan net.Conn),
		stopCh:      make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.bindPort, err)
	}
	s.listener = ln
	go s.acceptLoop()
	return nil
}

func (s *Server) listen() (net.Listener, error) {
	addr := fmt.Sprintf(":%d", s.bindPort)
	if !s.tls.Enabled {
		return net.Listen("tcp", addr)
	}
	if s.tls.CertFile == "" || s.tls.KeyFile == "" {
		return nil, fmt.Errorf("tls cert_file and key_file are required when tls is enabled")
	}
	if s.tls.CAFile == "" {
		return nil, fmt.Errorf("tls ca_file is required when tls is enabled")
	}
	cert, err := tls.LoadX509KeyPair(s.tls.CertFile, s.tls.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load tls certificate: %w", err)
	}
	caCert, err := os.ReadFile(s.tls.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tls ca_file: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if ok := clientCAs.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append tls ca_file to client cert pool")
	}
	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	}
	return tls.Listen("tcp", addr, config)
}

func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		if s.listener != nil {
			_ = s.listener.Close()
		}
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

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Errorf("Accept failed: %v", err)
				continue
			}
		}
		go s.handleIncoming(conn)
	}
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

	if loginMsg.Token != s.token {
		s.logger.Warnf("Authentication failed [%s]: token mismatch", conn.RemoteAddr())
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "authentication failed"})
		_ = conn.Close()
		return
	}

	sess := &ControlSession{
		runID:  uuid.New().String(),
		conn:   conn,
		stopCh: make(chan struct{}),
	}

	s.mu.Lock()
	s.clients[sess.runID] = sess
	s.mu.Unlock()

	s.logger.Infof("Client connected [%s], runID=%s", conn.RemoteAddr(), sess.runID)

	defer s.removeClient(sess.runID)

	if err := sess.WriteMsg(utils.MsgLoginResp, utils.LoginRespMsg{RunID: sess.runID}); err != nil {
		s.logger.Errorf("Failed to send LoginResp: %v", err)
		return
	}

	for {
		msgType, payload, err := utils.ReadMsg(sess.conn)
		if err != nil {
			s.logger.Infof("Client control connection disconnected runID=%s: %v", sess.runID, err)
			return
		}
		switch msgType {
		case utils.MsgNewProxy:
			s.handleNewProxy(sess, payload)
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

func (s *Server) handleNewProxy(sess *ControlSession, payload []byte) {

	var msg utils.NewProxyMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.sendProxyResp(sess, "", "failed to parse NewProxyMsg")
		return
	}

	if msg.Type != "tcp" {
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("unsupported proxy type: %s", msg.Type))
		return
	}

	if msg.RemotePort <= 0 || msg.RemotePort > 65535 {
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("invalid remote port: %d", msg.RemotePort))
		return
	}

	entry := &proxyEntry{
		name:       msg.Name,
		remotePort: msg.RemotePort,
		runID:      sess.runID,
	}

	s.mu.Lock()
	if _, exists := s.proxies[msg.Name]; exists {
		s.mu.Unlock()
		s.sendProxyResp(sess, msg.Name, "proxy name already exists")
		return
	}
	s.proxies[msg.Name] = entry
	s.mu.Unlock()

	registered := false
	defer func() {
		if registered {
			return
		}
		s.mu.Lock()
		if current, ok := s.proxies[msg.Name]; ok && current == entry {
			delete(s.proxies, msg.Name)
		}
		s.mu.Unlock()
	}()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", msg.RemotePort))
	if err != nil {
		s.logger.Errorf("Failed to listen on remote port %d: %v", msg.RemotePort, err)
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("failed to listen on port %d: %v", msg.RemotePort, err))
		return
	}

	s.mu.Lock()
	current, ok := s.proxies[msg.Name]
	if !ok || current != entry {
		s.mu.Unlock()
		_ = ln.Close()
		return
	}
	entry.listener = ln
	s.mu.Unlock()
	registered = true

	s.logger.Infof("Proxy registered successfully: name=%s, remotePort=%d, runID=%s", msg.Name, msg.RemotePort, sess.runID)
	s.sendProxyResp(sess, msg.Name, "")

	go s.proxyAcceptLoop(entry, sess)

}

func (s *Server) sendProxyResp(sess *ControlSession, name, errMsg string) {
	if err := sess.WriteMsg(utils.MsgNewProxyResp, utils.NewProxyRespMsg{
		Name:  name,
		Error: errMsg,
	}); err != nil {
		s.logger.Errorf("Failed to send NewProxyResp runID=%s proxy=%s: %v", sess.runID, name, err)
	}
}

func (s *Server) proxyAcceptLoop(entry *proxyEntry, sess *ControlSession) {

	defer func() {
		_ = entry.listener.Close()
		s.mu.Lock()
		if current, ok := s.proxies[entry.name]; ok && current == entry {
			delete(s.proxies, entry.name)
		}
		s.mu.Unlock()
		s.logger.Infof("Proxy stopped: name=%s", entry.name)
	}()

	for {
		userConn, err := entry.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			case <-sess.stopCh:
				return
			default:
				s.logger.Errorf("Proxy [%s] Accept failed: %v", entry.name, err)
				return
			}
		}
		allowed, srcIP, reason := s.ipFilter.AllowIP(userConn.RemoteAddr())
		if !allowed {
			s.logger.Warnf("User connection rejected by ip filter: proxy=%s, src=%s, ip=%s, reason=%s", entry.name, userConn.RemoteAddr(), srcIP, reason)
			_ = userConn.Close()
			continue
		}
		s.logger.Infof("User connection arrived: proxy=%s, src=%s, ip=%s", entry.name, userConn.RemoteAddr(), srcIP)
		go s.bridgeUserConn(userConn, entry, sess)
	}

}

func (s *Server) bridgeUserConn(userConn net.Conn, entry *proxyEntry, sess *ControlSession) {

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
		ProxyName: entry.name,
	}
	if err := sess.WriteMsg(utils.MsgNewWorkConn, msg); err != nil {
		s.logger.Errorf("Failed to send NewWorkConn: %v", err)
		return
	}

	select {
	case workConn := <-workCh:
		s.logger.Debugf("Work connection ready: workID=%s, proxy=%s", workID, entry.name)
		utils.Pipe(userConn, workConn)
	case <-time.After(10 * time.Second):
		s.logger.Warnf("Timed out waiting for work connection: workID=%s, proxy=%s", workID, entry.name)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.clients[runID]
	if ok {
		close(sess.stopCh)
		_ = sess.conn.Close()
		delete(s.clients, runID)
	}
	for name, proxy := range s.proxies {
		if proxy.runID == runID {
			if proxy.listener != nil {
				_ = proxy.listener.Close()
			}
			delete(s.proxies, name)
			s.logger.Infof("Proxy removed: name=%s (client disconnected)", name)
		}
	}
	s.logger.Infof("Client session cleaned up: runID=%s", runID)
}
