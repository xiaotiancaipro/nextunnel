package services

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type Client struct {
	Config   *configs.Client
	Proxies  []configs.Proxy
	Logger   *zap.Logger
	DialWork func() (net.Conn, error)
}

func (s *Client) Login(conn net.Conn) error {
	payload := sharedprotocol.LoginMsg{
		Id: s.Config.Id,
	}
	if err := sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLogin, payload); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to write LoginMsg: %v", err))
		return fmt.Errorf("failed to send LoginMsg")
	}
	return nil
}

func (s *Client) LoginResponse(conn net.Conn) (string, error) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to read LoginResp: %v", err))
		return "", fmt.Errorf("failed to read LoginResp")
	}
	if msgType != sharedprotocol.MsgLoginResp {
		s.Logger.Error(fmt.Sprintf("Invalid LoginResp msg type: %v", msgType))
		return "", fmt.Errorf("expected LoginResp")
	}

	var loginResp sharedprotocol.LoginRespMsg
	if err := sharedprotocol.Decode(payload, &loginResp); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse LoginResp: %v", err))
		return "", fmt.Errorf("failed to parse LoginResp")
	}
	if loginResp.Error != "" {
		s.Logger.Error(fmt.Sprintf("Login rejected by server: %v", loginResp.Error))
		return "", fmt.Errorf("login rejected by server")
	}

	return loginResp.RunID, nil
}

func (s *Client) ProxiesApply(conn net.Conn) error {
	proxies := make([]sharedprotocol.ProxiesApplyMsgItem, 0, len(s.Proxies))
	for _, proxy := range s.Proxies {
		proxies = append(proxies, sharedprotocol.ProxiesApplyMsgItem{
			Name:       proxy.Name,
			Type:       proxy.Type,
			RemotePort: proxy.RemotePort,
			LocalIP:    proxy.LocalIP,
			LocalPort:  proxy.LocalPort,
		})
	}

	payload := sharedprotocol.ProxiesApplyMsg{
		Proxies: proxies,
	}
	if err := sharedprotocol.WriteMsg(conn, sharedprotocol.MsgProxiesApply, payload); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to write ProxiesApplyMsg: %v", err))
		return fmt.Errorf("failed to send ProxiesApplyMsg")
	}
	return nil
}

func (s *Client) ProxiesApplyResponse(conn net.Conn) error {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to read ProxiesApplyResp: %v", err))
		return fmt.Errorf("failed to read ProxiesApplyResp")
	}
	if msgType != sharedprotocol.MsgProxiesApplyResp {
		s.Logger.Error(fmt.Sprintf("Invalid ProxiesApplyResp msg type: %v", msgType))
		return fmt.Errorf("expected ProxiesApplyResp")
	}

	var resp sharedprotocol.ProxiesApplyRespMsg
	if err := sharedprotocol.Decode(payload, &resp); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse ProxiesApplyResp: %v", err))
		return fmt.Errorf("failed to parse ProxiesApplyResp")
	}
	if resp.Error != "" {
		s.Logger.Error(fmt.Sprintf("ProxiesApply rejected by server: %v", resp.Error))
		return fmt.Errorf("proxies apply rejected by server")
	}

	for _, proxy := range s.Proxies {
		s.Logger.Info(fmt.Sprintf("Proxy applied successfully: name=%s", proxy.Name))
	}
	return nil
}

func (s *Client) WorkConn(msg sharedprotocol.NewWorkConnMsg) {
	proxy := s.FindProxy(msg.ProxyName)
	if proxy == nil {
		s.Logger.Error(fmt.Sprintf("Received work connection request for unknown proxy: %s", msg.ProxyName))
		return
	}

	if s.DialWork == nil {
		s.Logger.Error("DialWork is not configured; cannot open work channel")
		return
	}

	workTLS, err := s.DialWork()
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to dial work TLS connection: %v", err))
		return
	}

	payload := sharedprotocol.StartWorkConnMsg{WorkID: msg.WorkID}
	if err := sharedprotocol.WriteMsg(workTLS, sharedprotocol.MsgStartWorkConn, payload); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to send StartWorkConn: %v", err))
		_ = workTLS.Close()
		return
	}

	localAddr := net.JoinHostPort(proxy.LocalIP, strconv.Itoa(proxy.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to connect to local service [%s -> %s]: %v", msg.ProxyName, localAddr, err))
		_ = workTLS.Close()
		return
	}
	s.Logger.Info(fmt.Sprintf("Work connection bridged: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr))

	network.Pipe(workTLS, localConn)
}

func (s *Client) FindProxy(name string) *configs.Proxy {
	for i := range s.Proxies {
		if s.Proxies[i].Name == name {
			return &s.Proxies[i]
		}
	}
	return nil
}
