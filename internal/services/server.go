package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
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

func (s *Server) ProxiesApply(conn net.Conn, payload []byte, clientIdP *string) error {

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

	// TODO

	_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("Client config applied: clientID=%s", *clientIdP))
	return nil

}
