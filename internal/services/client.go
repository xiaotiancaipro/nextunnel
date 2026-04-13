package services

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

type ProxyConfig struct {
	Name       string
	Type       string // 当前仅支持 "tcp"
	LocalIP    string
	LocalPort  int
	RemotePort int
}

type ClientParams struct {
	ServerAddr string
	ServerPort int
	Token      string
	Proxies    []ProxyConfig
	Logger     *logrus.Logger
}

type Client struct {
	serverAddr string
	serverPort int
	token      string
	proxies    []ProxyConfig
	logger     *logrus.Logger
	runID      string
	ctrlCon    net.Conn
	mu         sync.Mutex
	stopCh     chan struct{}
}

type msgChan struct {
	msgType byte
	payload []byte
	err     error
}

func NewClient(p *ClientParams) (*Client, error) {
	if p.ServerAddr == "" {
		return nil, fmt.Errorf("服务端地址不能为空")
	}
	if p.ServerPort <= 0 || p.ServerPort > 65535 {
		return nil, fmt.Errorf("无效的服务端端口: %d", p.ServerPort)
	}
	return &Client{
		serverAddr: p.ServerAddr,
		serverPort: p.ServerPort,
		token:      p.Token,
		proxies:    p.Proxies,
		logger:     p.Logger,
		stopCh:     make(chan struct{}),
	}, nil
}

func (c *Client) Start() error {
	if err := c.connect(); err != nil {
		return err
	}
	go c.reconnectLoop()
	return nil
}

func (c *Client) Stop() {
	close(c.stopCh)
	c.mu.Lock()
	if c.ctrlCon != nil {
		_ = c.ctrlCon.Close()
	}
	c.mu.Unlock()
}

func (c *Client) serverAddrStr() string {
	return fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort)
}

func (c *Client) connect() error {

	conn, err := net.DialTimeout("tcp", c.serverAddrStr(), 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接服务端失败: %w", err)
	}

	if err := utils.WriteMsg(conn, utils.MsgLogin, utils.LoginMsg{Token: c.token}); err != nil {
		_ = conn.Close()
		return fmt.Errorf("发送 LoginMsg 失败: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("读取 LoginResp 失败: %w", err)
	}
	if msgType != utils.MsgLoginResp {
		_ = conn.Close()
		return fmt.Errorf("期望 LoginResp，收到 0x%02x", msgType)
	}

	var loginResp utils.LoginRespMsg
	if err := utils.Decode(payload, &loginResp); err != nil {
		_ = conn.Close()
		return fmt.Errorf("解析 LoginResp 失败: %w", err)
	}
	if loginResp.Error != "" {
		_ = conn.Close()
		return fmt.Errorf("登录失败: %s", loginResp.Error)
	}

	c.mu.Lock()
	c.ctrlCon = conn
	c.runID = loginResp.RunID
	c.mu.Unlock()

	c.logger.Infof("登录成功, runID=%s", loginResp.RunID)

	for _, proxy := range c.proxies {
		if err := c.registerProxy(conn, proxy); err != nil {
			c.logger.Errorf("注册代理失败 [%s]: %v", proxy.Name, err)
		}
	}

	// 启动控制循环（心跳 + 处理 NewWorkConn）
	go c.controlLoop(conn)

	return nil

}

func (c *Client) registerProxy(conn net.Conn, proxy ProxyConfig) error {

	msg := utils.NewProxyMsg{
		Name:       proxy.Name,
		Type:       proxy.Type,
		RemotePort: proxy.RemotePort,
	}
	if err := utils.WriteMsg(conn, utils.MsgNewProxy, msg); err != nil {
		return fmt.Errorf("发送 NewProxyMsg 失败: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("读取 NewProxyResp 失败: %w", err)
	}
	if msgType != utils.MsgNewProxyResp {
		return fmt.Errorf("期望 NewProxyResp，收到 0x%02x", msgType)
	}

	var resp utils.NewProxyRespMsg
	if err := utils.Decode(payload, &resp); err != nil {
		return fmt.Errorf("解析 NewProxyResp 失败: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("服务端拒绝代理: %s", resp.Error)
	}

	c.logger.Infof("代理注册成功: name=%s, remotePort=%d → %s:%d", proxy.Name, proxy.RemotePort, proxy.LocalIP, proxy.LocalPort)
	return nil

}

func (c *Client) controlLoop(conn net.Conn) {

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	msgCh := make(chan msgChan, 1)

	go func() {
		for {
			msgType, payload, err := utils.ReadMsg(conn)
			msgCh <- msgChan{msgType, payload, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-c.stopCh:
			return
		case <-heartbeatTicker.C:
			c.mu.Lock()
			_ = utils.WriteMsg(conn, utils.MsgPing, utils.PingMsg{})
			c.mu.Unlock()
		case result := <-msgCh:
			if result.err != nil {
				c.logger.Errorf("读取控制消息失败: %v", result.err)
				return
			}
			switch result.msgType {
			case utils.MsgNewWorkConn:
				var msg utils.NewWorkConnMsg
				if err := utils.Decode(result.payload, &msg); err != nil {
					c.logger.Errorf("解析 NewWorkConnMsg 失败: %v", err)
					continue
				}
				go c.handleWorkConn(msg)
			case utils.MsgPong:
			default:
				c.logger.Warnf("收到未知控制消息 0x%02x", result.msgType)
			}
		}
	}

}

func (c *Client) handleWorkConn(msg utils.NewWorkConnMsg) {

	proxy := c.findProxy(msg.ProxyName)
	if proxy == nil {
		c.logger.Errorf("收到未知代理的工作连接请求: %s", msg.ProxyName)
		return
	}

	workConn, err := net.DialTimeout("tcp", c.serverAddrStr(), 10*time.Second)
	if err != nil {
		c.logger.Errorf("建立工作连接失败 [%s]: %v", msg.ProxyName, err)
		return
	}

	// 发送 StartWorkConn 告知服务端此连接对应的 workID
	if err := utils.WriteMsg(workConn, utils.MsgStartWorkConn, utils.StartWorkConnMsg{WorkID: msg.WorkID}); err != nil {
		c.logger.Errorf("发送 StartWorkConn 失败: %v", err)
		_ = workConn.Close()
		return
	}

	// 连接本地服务
	localAddr := fmt.Sprintf("%s:%d", proxy.LocalIP, proxy.LocalPort)
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		c.logger.Errorf("连接本地服务失败 [%s → %s]: %v", msg.ProxyName, localAddr, err)
		_ = workConn.Close()
		return
	}

	c.logger.Debugf("工作连接桥接: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr)

	// 双向转发数据
	utils.Pipe(workConn, localConn)

}

func (c *Client) findProxy(name string) *ProxyConfig {
	for i := range c.proxies {
		if c.proxies[i].Name == name {
			return &c.proxies[i]
		}
	}
	return nil
}

func (c *Client) reconnectLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		case <-time.After(5 * time.Second):
			c.mu.Lock()
			conn := c.ctrlCon
			c.mu.Unlock()
			if conn == nil {
				continue
			}
			if err := conn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
				c.logger.Infof("控制连接已断开，尝试重连...")
				c.tryReconnect()
			}
			_ = conn.SetDeadline(time.Time{})
		}
	}
}

func (c *Client) tryReconnect() {
	backoff := 2 * time.Second
	maxBackoff := 60 * time.Second
	for {
		select {
		case <-c.stopCh:
			return
		case <-time.After(backoff):
			c.logger.Infof("正在重连服务端 %s ...", c.serverAddrStr())
			if err := c.connect(); err != nil {
				c.logger.Errorf("重连失败: %v, 将在 %v 后重试", err, backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			c.logger.Infof("重连成功")
			return
		}
	}
}
