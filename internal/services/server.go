package services

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
)

type Server struct {
	Config     *configs.Server
	IpBlackMap map[string]bool
	Logger     *zap.Logger
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
		go s.ProxyAcceptLoop(name, listener, stopCh)
	}

	_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("Client config applied: clientID=%s", *clientIdP))
	return nil

}

func (s *Server) ProxyAcceptLoop(proxyName string, listener net.Listener, stopCh chan struct{}) {

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
		if err != nil {
			s.Logger.Warn(fmt.Sprintf("User connection rejected by ip filter: proxy=%s, ip=%s, reason=%s", proxyName, *ipP, err.Error()))
			_ = conn.Close()
			continue
		}

		s.Logger.Info(fmt.Sprintf("User connection arrived: proxy=%s, ip=%s", proxyName, *ipP))

		go s.BridgeClientConn(conn, proxyName, stopCh)

	}

}

func (s *Server) AllowIP(addr net.Addr) (*string, error) {

	if len(s.IpBlackMap) == 0 {
		return nil, nil
	}

	host := addr.String()
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	ipP, err := utils.NormalizeIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote ip")
	}

	if _, ok := s.IpBlackMap[*ipP]; ok {
		return ipP, fmt.Errorf("matched deny list")
	}

	return ipP, nil

}

func (s *Server) BridgeClientConn(conn net.Conn, proxyName string, stopCh chan struct{}) {

	defer func() { _ = conn.Close() }()

	workID := uuid.New().String()
	msg := utils.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: proxyName,
	}
	if err := utils.WriteMsg(conn, utils.MsgNewWorkConn, msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to send NewWorkConn: %v", err))
		return
	}

	workCh := make(chan net.Conn, 1)
	select {
	case <-stopCh:
		return
	case workConn := <-workCh:
		s.Pipe(conn, workConn)
	case <-time.After(10 * time.Second):
		s.Logger.Warn(fmt.Sprintf("Timed out waiting for work connection: workID=%s, proxy=%s", workID, proxyName))
	}

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

func (s *Server) StartWorkConn(payload []byte) error {
	var msg utils.StartWorkConnMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse StartWorkConnMsg: %v", err))
		return fmt.Errorf("failed to parse StartWorkConnMsg")
	}
	_ = msg.WorkID // TODO work_id verification
	return nil
}
