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
	Type       string // currently only "tcp" is supported
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
		return nil, fmt.Errorf("server address cannot be empty")
	}
	if p.ServerPort <= 0 || p.ServerPort > 65535 {
		return nil, fmt.Errorf("invalid server port: %d", p.ServerPort)
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
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := utils.WriteMsg(conn, utils.MsgLogin, utils.LoginMsg{Token: c.token}); err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to send LoginMsg: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to read LoginResp: %w", err)
	}
	if msgType != utils.MsgLoginResp {
		_ = conn.Close()
		return fmt.Errorf("expected LoginResp, got 0x%02x", msgType)
	}

	var loginResp utils.LoginRespMsg
	if err := utils.Decode(payload, &loginResp); err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to parse LoginResp: %w", err)
	}
	if loginResp.Error != "" {
		_ = conn.Close()
		return fmt.Errorf("login failed: %s", loginResp.Error)
	}

	c.mu.Lock()
	c.ctrlCon = conn
	c.runID = loginResp.RunID
	c.mu.Unlock()

	c.logger.Infof("Login successful, runID=%s", loginResp.RunID)

	for _, proxy := range c.proxies {
		if err := c.registerProxy(conn, proxy); err != nil {
			c.logger.Errorf("Failed to register proxy [%s]: %v", proxy.Name, err)
		}
	}

	go c.controlLoop(conn) // start control loop (heartbeat + handle NewWorkConn)

	return nil

}

func (c *Client) registerProxy(conn net.Conn, proxy ProxyConfig) error {

	msg := utils.NewProxyMsg{
		Name:       proxy.Name,
		Type:       proxy.Type,
		RemotePort: proxy.RemotePort,
	}
	if err := utils.WriteMsg(conn, utils.MsgNewProxy, msg); err != nil {
		return fmt.Errorf("failed to send NewProxyMsg: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("failed to read NewProxyResp: %w", err)
	}
	if msgType != utils.MsgNewProxyResp {
		return fmt.Errorf("expected NewProxyResp, got 0x%02x", msgType)
	}

	var resp utils.NewProxyRespMsg
	if err := utils.Decode(payload, &resp); err != nil {
		return fmt.Errorf("failed to parse NewProxyResp: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("proxy rejected by server: %s", resp.Error)
	}

	c.logger.Infof("Proxy registered successfully: name=%s, remotePort=%d → %s:%d", proxy.Name, proxy.RemotePort, proxy.LocalIP, proxy.LocalPort)
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
				c.logger.Errorf("Failed to read control message: %v", result.err)
				return
			}
			switch result.msgType {
			case utils.MsgNewWorkConn:
				var msg utils.NewWorkConnMsg
				if err := utils.Decode(result.payload, &msg); err != nil {
					c.logger.Errorf("Failed to parse NewWorkConnMsg: %v", err)
					continue
				}
				go c.handleWorkConn(msg)
			case utils.MsgPong:
			default:
				c.logger.Warnf("Received unknown control message 0x%02x", result.msgType)
			}
		}
	}

}

func (c *Client) handleWorkConn(msg utils.NewWorkConnMsg) {

	proxy := c.findProxy(msg.ProxyName)
	if proxy == nil {
		c.logger.Errorf("Received work connection request for unknown proxy: %s", msg.ProxyName)
		return
	}

	workConn, err := net.DialTimeout("tcp", c.serverAddrStr(), 10*time.Second)
	if err != nil {
		c.logger.Errorf("Failed to establish work connection [%s]: %v", msg.ProxyName, err)
		return
	}

	// send StartWorkConn to inform server of this connection's workID
	if err := utils.WriteMsg(workConn, utils.MsgStartWorkConn, utils.StartWorkConnMsg{WorkID: msg.WorkID}); err != nil {
		c.logger.Errorf("Failed to send StartWorkConn: %v", err)
		_ = workConn.Close()
		return
	}

	// connect to local service
	localAddr := fmt.Sprintf("%s:%d", proxy.LocalIP, proxy.LocalPort)
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		c.logger.Errorf("Failed to connect to local service [%s → %s]: %v", msg.ProxyName, localAddr, err)
		_ = workConn.Close()
		return
	}

	c.logger.Debugf("Work connection bridged: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr)

	// bidirectional data forwarding
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
				c.logger.Infof("Control connection disconnected, attempting reconnect...")
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
			c.logger.Infof("Reconnecting to server %s ...", c.serverAddrStr())
			if err := c.connect(); err != nil {
				c.logger.Errorf("Reconnect failed: %v, will retry in %v", err, backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			c.logger.Infof("Reconnect successful")
			return
		}
	}
}
