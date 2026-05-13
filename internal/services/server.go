package services

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
)

type Server struct {
	Config       *configs.Server
	IpBlackMap   map[string]bool
	Logger       *zap.Logger
	pendingMu    sync.Mutex
	pendingWorks map[string]net.Conn
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

func (s *Server) Login(conn net.Conn, payload []byte) (*string, *string, error) {

	var loginMsg utils.LoginMsg
	if err := utils.Decode(payload, &loginMsg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse LoginMsg: %v", err))
		return nil, nil, fmt.Errorf("failed to parse LoginMsg")
	}

	if loginMsg.Id == "" {
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "client_id cannot be empty"})
		return nil, nil, fmt.Errorf("client_id is empty")
	}

	if loginMsg.Token != s.Config.Token {
		s.Logger.Warn(fmt.Sprintf("Authentication failed [%s]: token mismatch", conn.RemoteAddr()))
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "authentication failed"})
		return nil, nil, fmt.Errorf("authentication failed")
	}

	runID := uuid.New().String()
	if err := utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{RunID: runID}); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to send LoginResp: %v", err))
		return nil, nil, fmt.Errorf("failed to send LoginResp")
	}

	return &loginMsg.Id, &runID, nil

}

func (s *Server) ProxiesApply(conn net.Conn, payload []byte, clientIdP *string, stopCh chan struct{}) error {

	var msg utils.ProxiesApplyMsg
	if err := utils.Decode(payload, &msg); err != nil {
		e := fmt.Sprintf("failed to parse ApplyConfigMsg: %v", err)
		_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
		s.Logger.Error(e)
		return fmt.Errorf("failed to parse ApplyConfigMsg")
	}

	desired := make(map[string]utils.ProxiesApplyMsgItem, len(msg.Proxies))
	for _, proxy := range msg.Proxies {
		if proxy.Name == "" {
			e := fmt.Sprintf("[%s]Proxy name is empty", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy name is empty")
		}
		if proxy.Type != "tcp" {
			e := fmt.Sprintf("[%s]Proxy type is invalid", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy type is invalid")
		}
		if proxy.RemotePort <= 0 || proxy.RemotePort > 65535 {
			e := fmt.Sprintf("[%s]Proxy remote port is invalid", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy remote port is invalid")
		}
		if _, exists := desired[proxy.Name]; exists {
			e := fmt.Sprintf("[%s]Proxy name is duplicated", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy name is duplicated")
		}
		desired[proxy.Name] = proxy
	}

	opened := make(map[string]net.Listener)
	openedClose := func() {
		for _, ln := range opened {
			_ = ln.Close()
		}
	}
	for _, proxy := range msg.Proxies {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", proxy.RemotePort))
		if err != nil {
			openedClose()
			e := fmt.Sprintf("Failed to listen on port %d: %v", proxy.RemotePort, err)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("failed to listen on port %d", proxy.RemotePort)
		}
		opened[proxy.Name] = ln
	}

	for name, listener := range opened {
		go s.ProxyAcceptLoop(conn, name, listener, stopCh)
	}

	_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("Client config applied: clientID=%s", *clientIdP))
	return nil

}

func (s *Server) ProxyAcceptLoop(controlConn net.Conn, proxyName string, listener net.Listener, stopCh chan struct{}) {

	defer func() {
		_ = listener.Close()
		s.Logger.Info(fmt.Sprintf("Proxy stopped: name=%s", proxyName))
	}()

	for {

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-stopCh:
				return
			default:
				s.Logger.Error(fmt.Sprintf("Proxy [%s] accept loop exiting: %v", proxyName, err))
				return
			}
		}

		ipP, err := s.AllowIP(conn.RemoteAddr())
		ip := "(unknown)"
		if ipP != nil {
			ip = *ipP
		}
		if err != nil {
			s.Logger.Warn(fmt.Sprintf("User connection rejected by ip filter: proxy=%s, ip=%s, reason=%s", proxyName, ip, err.Error()))
			_ = conn.Close()
			continue
		}

		s.Logger.Info(fmt.Sprintf("User connection arrived: proxy=%s, ip=%s", proxyName, ip))

		go s.BridgeClientConn(controlConn, conn, proxyName, stopCh)

	}

}

func (s *Server) AllowIP(addr net.Addr) (*string, error) {

	host := addr.String()
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	ipP, err := utils.NormalizeIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote ip")
	}

	if len(s.IpBlackMap) > 0 {
		if _, ok := s.IpBlackMap[*ipP]; ok {
			return ipP, fmt.Errorf("matched deny list")
		}
	}

	return ipP, nil

}

func (s *Server) BridgeClientConn(controlConn, conn net.Conn, proxyName string, stopCh chan struct{}) {

	workID := uuid.New().String()
	s.RegisterPendingWork(workID, conn)

	select {
	case <-stopCh:
		if c := s.TakePendingConn(workID); c != nil {
			_ = c.Close()
		}
		return
	default:
	}

	msg := utils.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: proxyName,
	}
	if err := utils.WriteMsg(controlConn, utils.MsgNewWorkConn, msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to notify client (NewWorkConn): %v", err))
		if c := s.TakePendingConn(workID); c != nil {
			_ = c.Close()
		}
		return
	}

}

func (s *Server) RegisterPendingWork(workID string, conn net.Conn) {

	s.pendingMu.Lock()
	if s.pendingWorks == nil {
		s.pendingWorks = make(map[string]net.Conn)
	}
	s.pendingMu.Unlock()

	s.pendingMu.Lock()
	s.pendingWorks[workID] = conn
	s.pendingMu.Unlock()

	time.AfterFunc(15*time.Second, func() {
		s.pendingMu.Lock()
		c, ok := s.pendingWorks[workID]
		if ok {
			delete(s.pendingWorks, workID)
		}
		s.pendingMu.Unlock()
		if ok {
			_ = c.Close()
			s.Logger.Warn(fmt.Sprintf("Timed out waiting for work channel; closed user connection: workID=%s", workID))
		}
	})

}

func (s *Server) StartWorkConn(workTLS net.Conn, payload []byte) error {
	var msg utils.StartWorkConnMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse StartWorkConnMsg: %v", err))
		return fmt.Errorf("failed to parse StartWorkConnMsg")
	}
	if msg.WorkID == "" {
		_ = workTLS.Close()
		return fmt.Errorf("work_id is empty")
	}
	userConn := s.TakePendingConn(msg.WorkID)
	if userConn == nil {
		s.Logger.Warn(fmt.Sprintf("No pending user connection for work_id=%s", msg.WorkID))
		_ = workTLS.Close()
		return fmt.Errorf("unknown or expired work_id")
	}
	go s.Pipe(userConn, workTLS)
	return nil
}

func (s *Server) TakePendingConn(workID string) net.Conn {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingWorks == nil {
		return nil
	}
	c, ok := s.pendingWorks[workID]
	if !ok {
		return nil
	}
	delete(s.pendingWorks, workID)
	return c
}

func (s *Server) Pipe(a, b net.Conn) {
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	done := make(chan struct{}, 2)
	copyFn := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyFn(a, b)
	go copyFn(b, a)
	<-done
}
