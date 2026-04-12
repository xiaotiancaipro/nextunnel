package services

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

// ClientParams Client 初始化参数
type ClientParams struct {
	ServerAddr string
	ServerPort int
	Token      string
	Proxies    []ProxyConfig
	Logger     *logrus.Logger
}

// ProxyConfig 客户端代理配置
type ProxyConfig struct {
	Name       string
	Type       string // 当前仅支持 "tcp"
	RemotePort int
	LocalIP    string
	LocalPort  int
}

// Client 内网穿透客户端核心
type Client struct {
	serverAddr string
	serverPort int
	token      string
	proxies    []ProxyConfig
	logger     *logrus.Logger

	runID   string
	ctrlCon net.Conn
	mu      sync.Mutex

	stopCh chan struct{}
}

// NewClient 创建客户端实例
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

// Start 连接服务端并启动控制循环（带断线重连）
func (c *Client) Start() error {
	if err := c.connect(); err != nil {
		return err
	}
	go c.reconnectLoop()
	return nil
}

// Stop 停止客户端
func (c *Client) Stop() {
	close(c.stopCh)
	c.mu.Lock()
	if c.ctrlCon != nil {
		_ = c.ctrlCon.Close()
	}
	c.mu.Unlock()
}

// serverAddr 返回服务端地址字符串
func (c *Client) serverAddrStr() string {
	return fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort)
}

// connect 建立控制连接并完成认证、代理注册
func (c *Client) connect() error {
	conn, err := net.DialTimeout("tcp", c.serverAddrStr(), 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接服务端失败: %w", err)
	}

	// 发送登录消息
	if err := utils.WriteMsg(conn, utils.MsgLogin, utils.LoginMsg{Token: c.token}); err != nil {
		_ = conn.Close()
		return fmt.Errorf("发送 LoginMsg 失败: %w", err)
	}

	// 读取登录响应
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

	// 注册所有代理
	for _, proxy := range c.proxies {
		if err := c.registerProxy(conn, proxy); err != nil {
			c.logger.Errorf("注册代理失败 [%s]: %v", proxy.Name, err)
		}
	}

	// 启动控制循环（心跳 + 处理 NewWorkConn）
	go c.controlLoop(conn)
	return nil
}

// registerProxy 向服务端注册一个 TCP 代理
func (c *Client) registerProxy(conn net.Conn, proxy ProxyConfig) error {
	if err := utils.WriteMsg(conn, utils.MsgNewProxy, utils.NewProxyMsg{
		Name:       proxy.Name,
		Type:       proxy.Type,
		RemotePort: proxy.RemotePort,
	}); err != nil {
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

	c.logger.Infof("代理注册成功: name=%s, remotePort=%d → %s:%d",
		proxy.Name, proxy.RemotePort, proxy.LocalIP, proxy.LocalPort)
	return nil
}

// controlLoop 持续读取服务端控制消息，处理工作连接请求和心跳
func (c *Client) controlLoop(conn net.Conn) {
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	msgCh := make(chan struct {
		msgType byte
		payload []byte
		err     error
	}, 1)

	go func() {
		for {
			msgType, payload, err := utils.ReadMsg(conn)
			msgCh <- struct {
				msgType byte
				payload []byte
				err     error
			}{msgType, payload, err}
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
				// 心跳响应，忽略
			default:
				c.logger.Warnf("收到未知控制消息 0x%02x", result.msgType)
			}
		}
	}
}

// handleWorkConn 响应服务端的工作连接请求
// 1. 向服务端建立新的 TCP 连接
// 2. 发送 StartWorkConn 消息
// 3. 拨号本地服务
// 4. 双向桥接数据
func (c *Client) handleWorkConn(msg utils.NewWorkConnMsg) {
	proxy := c.findProxy(msg.ProxyName)
	if proxy == nil {
		c.logger.Errorf("收到未知代理的工作连接请求: %s", msg.ProxyName)
		return
	}

	// 建立到服务端的工作连接
	workConn, err := net.DialTimeout("tcp", c.serverAddrStr(), 10*time.Second)
	if err != nil {
		c.logger.Errorf("建立工作连接失败 [%s]: %v", msg.ProxyName, err)
		return
	}

	// 发送 StartWorkConn 告知服务端此连接对应的 workID
	if err := utils.WriteMsg(workConn, utils.MsgStartWorkConn, utils.StartWorkConnMsg{
		WorkID: msg.WorkID,
	}); err != nil {
		c.logger.Errorf("发送 StartWorkConn 失败: %v", err)
		_ = workConn.Close()
		return
	}

	// 拨号本地服务
	localAddr := fmt.Sprintf("%s:%d", proxy.LocalIP, proxy.LocalPort)
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		c.logger.Errorf("连接本地服务失败 [%s → %s]: %v", msg.ProxyName, localAddr, err)
		_ = workConn.Close()
		return
	}

	c.logger.Debugf("工作连接桥接: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr)

	// 双向转发数据
	pipe(workConn, localConn)
}

// findProxy 按名称查找代理配置
func (c *Client) findProxy(name string) *ProxyConfig {
	for i := range c.proxies {
		if c.proxies[i].Name == name {
			return &c.proxies[i]
		}
	}
	return nil
}

// reconnectLoop 断线后自动重连
func (c *Client) reconnectLoop() {
	// 等待控制连接断开
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

			// 尝试 ping，若连接已断则重连
			if err := conn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
				c.logger.Infof("控制连接已断开，尝试重连...")
				c.tryReconnect()
			}
			_ = conn.SetDeadline(time.Time{})
		}
	}
}

// tryReconnect 执行重连，带指数退避
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

// pipe 双向转发两个连接之间的数据
func pipe(a, b net.Conn) {
	defer a.Close()
	defer b.Close()

	done := make(chan struct{}, 2)
	copyFn := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyFn(a, b)
	go copyFn(b, a)
	<-done
}
