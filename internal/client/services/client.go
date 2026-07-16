package services

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type Client struct {
	Config   *configs.Client
	Proxies  []configs.Proxy
	Logger   *zap.Logger
	Conn     net.Conn
	DialWork func() (net.Conn, error)
}

func (c *Client) Login() error {
	payload := protocol.LoginMsg{
		Id: c.Config.Id,
	}
	if err := protocol.WriteMsg(c.Conn, protocol.MsgLogin, payload); err != nil {
		c.Logger.Error(fmt.Sprintf("failed to write login msg: %v", err))
		return fmt.Errorf("failed to send LoginMsg")
	}
	return nil
}

func (c *Client) LoginResponse() (*string, error) {

	_ = c.Conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := protocol.ReadMsg(c.Conn)
	_ = c.Conn.SetDeadline(time.Time{})
	if err != nil {
		c.Logger.Error(fmt.Sprintf("failed to read login msg: %v", err))
		return nil, fmt.Errorf("failed to read LoginResp")
	}
	if msgType != protocol.MsgLoginResp {
		c.Logger.Error(fmt.Sprintf("invalid login msg type: %v", msgType))
		return nil, fmt.Errorf("expected LoginResp")
	}

	var loginResp protocol.LoginRespMsg
	if err := protocol.Decode(payload, &loginResp); err != nil {
		c.Logger.Error(fmt.Sprintf("failed to decode LoginResp: %v", err))
		return nil, fmt.Errorf("failed to parse LoginResp")
	}
	if loginResp.Error != "" {
		c.Logger.Error(fmt.Sprintf("login response error: %v", loginResp.Error))
		return nil, fmt.Errorf("login error")
	}

	return &loginResp.RunID, nil

}

func (c *Client) ProxiesApply() error {

	proxies := make([]protocol.ProxiesApplyMsgItem, 0, len(c.Proxies))
	for _, proxy := range c.Proxies {
		proxies = append(proxies, protocol.ProxiesApplyMsgItem{
			Name:       proxy.Name,
			Type:       proxy.Type,
			RemotePort: proxy.RemotePort,
			LocalIP:    proxy.LocalIP,
			LocalPort:  proxy.LocalPort,
		})
	}

	payload := protocol.ProxiesApplyMsg{
		Proxies: proxies,
	}
	if err := protocol.WriteMsg(c.Conn, protocol.MsgProxiesApply, payload); err != nil {
		c.Logger.Error(fmt.Sprintf("failed to write proxies msg: %v", err))
		return fmt.Errorf("failed to send ApplyConfigMsg")
	}

	return nil

}

func (c *Client) ProxiesApplyResponse() error {

	_ = c.Conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := protocol.ReadMsg(c.Conn)
	_ = c.Conn.SetDeadline(time.Time{})
	if err != nil {
		c.Logger.Error(fmt.Sprintf("Failed to read proxies msg: %v", err))
		return fmt.Errorf("failed to read ApplyConfigResp")
	}
	if msgType != protocol.MsgProxiesApplyResp {
		c.Logger.Error(fmt.Sprintf("Invalid proxies msg type: %v", msgType))
		return fmt.Errorf("expected ApplyConfigResp")
	}

	var resp protocol.ProxiesApplyRespMsg
	if err := protocol.Decode(payload, &resp); err != nil {
		c.Logger.Error(fmt.Sprintf("Failed to decode ApplyConfigResp: %v", err))
		return fmt.Errorf("failed to parse ApplyConfigResp")
	}
	if resp.Error != "" {
		c.Logger.Error(fmt.Sprintf("Failed to parse ApplyConfigResp: %v", resp.Error))
		return fmt.Errorf("apply config rejected by server")
	}

	for _, proxy := range c.Proxies {
		c.Logger.Info(fmt.Sprintf("Proxy applied successfully: name=%s", proxy.Name))
	}
	return nil

}

func (c *Client) WorkConn(msg protocol.NewWorkConnMsg) {

	proxy := c.FindProxy(msg.ProxyName)
	if proxy == nil {
		c.Logger.Error(fmt.Sprintf("Received work connection request for unknown proxy: %s", msg.ProxyName))
		return
	}

	if c.DialWork == nil {
		c.Logger.Error("DialWork is not configured; cannot open work channel")
		return
	}

	workTLS, err := c.DialWork()
	if err != nil {
		c.Logger.Error(fmt.Sprintf("Failed to dial work TLS connection: %v", err))
		return
	}

	payload := protocol.StartWorkConnMsg{WorkID: msg.WorkID}
	if err := protocol.WriteMsg(workTLS, protocol.MsgStartWorkConn, payload); err != nil {
		c.Logger.Error(fmt.Sprintf("Failed to send StartWorkConn: %v", err))
		_ = workTLS.Close()
		return
	}

	localAddr := net.JoinHostPort(proxy.LocalIP, strconv.Itoa(proxy.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		c.Logger.Error(fmt.Sprintf("Failed to connect to local service [%s -> %s]: %v", msg.ProxyName, localAddr, err))
		_ = workTLS.Close()
		return
	}
	c.Logger.Info(fmt.Sprintf("Work connection bridged: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr))

	c.Pipe(workTLS, localConn)

}

func (c *Client) FindProxy(name string) *configs.Proxy {
	for i := range c.Proxies {
		if c.Proxies[i].Name == name {
			return &c.Proxies[i]
		}
	}
	return nil
}

func (c *Client) Pipe(a, b net.Conn) {
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
