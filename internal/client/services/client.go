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
		s.Logger.Error(fmt.Sprintf("failed to write login msg: %v", err))
		return fmt.Errorf("failed to send LoginMsg")
	}
	return nil
}

func (s *Client) LoginResponse(conn net.Conn) (string, error) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to read login response: %v", err))
		return "", fmt.Errorf("failed to read LoginResp")
	}
	if msgType != sharedprotocol.MsgLoginResp {
		s.Logger.Error(fmt.Sprintf("invalid login response msg type: 0x%02x", msgType))
		return "", fmt.Errorf("expected LoginResp")
	}

	var loginResp sharedprotocol.LoginRespMsg
	if err := sharedprotocol.Decode(payload, &loginResp); err != nil {
		s.Logger.Error(fmt.Sprintf("failed to parse login response: %v", err))
		return "", fmt.Errorf("failed to parse LoginResp")
	}
	if loginResp.Error != "" {
		s.Logger.Error(fmt.Sprintf("login rejected by server: %v", loginResp.Error))
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
		s.Logger.Error(fmt.Sprintf("failed to write proxies apply msg: %v", err))
		return fmt.Errorf("failed to send ProxiesApplyMsg")
	}
	return nil
}

func (s *Client) ProxiesApplyResponse(conn net.Conn) error {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to read proxies apply response: %v", err))
		return fmt.Errorf("failed to read ProxiesApplyResp")
	}
	if msgType != sharedprotocol.MsgProxiesApplyResp {
		s.Logger.Error(fmt.Sprintf("invalid proxies apply response msg type: 0x%02x", msgType))
		return fmt.Errorf("expected ProxiesApplyResp")
	}

	var resp sharedprotocol.ProxiesApplyRespMsg
	if err := sharedprotocol.Decode(payload, &resp); err != nil {
		s.Logger.Error(fmt.Sprintf("failed to parse proxies apply response: %v", err))
		return fmt.Errorf("failed to parse ProxiesApplyResp")
	}
	if resp.Error != "" {
		s.Logger.Error(fmt.Sprintf("proxies apply rejected by server: %v", resp.Error))
		return fmt.Errorf("proxies apply rejected by server")
	}

	for _, proxy := range s.Proxies {
		s.Logger.Info(fmt.Sprintf("proxy applied: name=%s, remote_port=%d, local=%s:%d", proxy.Name, proxy.RemotePort, proxy.LocalIP, proxy.LocalPort))
	}
	return nil
}

func (s *Client) WorkConn(msg sharedprotocol.NewWorkConnMsg) {
	proxy := s.FindProxy(msg.ProxyName)
	if proxy == nil {
		s.Logger.Error(fmt.Sprintf("received work connection for unknown proxy: name=%s, work_id=%s", msg.ProxyName, msg.WorkID))
		return
	}

	if s.DialWork == nil {
		s.Logger.Error("dial work is not configured; cannot open work channel")
		return
	}

	workTLS, err := s.DialWork()
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to dial work tls connection: proxy=%s, work_id=%s, err=%v", msg.ProxyName, msg.WorkID, err))
		return
	}

	payload := sharedprotocol.StartWorkConnMsg{WorkID: msg.WorkID}
	if err := sharedprotocol.WriteMsg(workTLS, sharedprotocol.MsgStartWorkConn, payload); err != nil {
		s.Logger.Error(fmt.Sprintf("failed to send start work conn: proxy=%s, work_id=%s, err=%v", msg.ProxyName, msg.WorkID, err))
		_ = workTLS.Close()
		return
	}

	localAddr := net.JoinHostPort(proxy.LocalIP, strconv.Itoa(proxy.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to connect to local service: proxy=%s, local=%s, err=%v", msg.ProxyName, localAddr, err))
		_ = workTLS.Close()
		return
	}
	s.Logger.Info(fmt.Sprintf("work connection bridged: proxy=%s, work_id=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr))

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
