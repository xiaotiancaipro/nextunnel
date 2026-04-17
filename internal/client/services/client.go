package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

const (
	controlHeartbeatInterval = 30 * time.Second
	controlWriteTimeout      = 10 * time.Second
	controlReadTimeout       = 90 * time.Second
	reconnectInitialBackoff  = 2 * time.Second
	reconnectMaxBackoff      = 60 * time.Second
)

type Client struct {
	config   configs.ClientConfigs
	logger   *logrus.Logger
	runID    string
	ctrlCon  net.Conn
	mu       sync.Mutex
	stopCh   chan struct{}
	stopOnce sync.Once
	retiring atomic.Bool
}

type Params struct {
	Config configs.ClientConfigs
	Logger *logrus.Logger
}

type msgChan struct {
	msgType byte
	payload []byte
	err     error
}

func NewClient(params *Params) (*Client, error) {
	if params.Config.ClientID == "" {
		return nil, fmt.Errorf("client id cannot be empty")
	}
	if params.Config.ServerAddr == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}
	if params.Config.ServerPort <= 0 || params.Config.ServerPort > 65535 {
		return nil, fmt.Errorf("invalid server port: %d", params.Config.ServerPort)
	}
	return &Client{
		config: params.Config,
		logger: params.Logger,
		stopCh: make(chan struct{}),
	}, nil
}

func (c *Client) Start() error {
	return c.connect()
}

func (c *Client) Retire() {
	c.retiring.Store(true)
}

func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.ctrlCon != nil {
			_ = c.ctrlCon.Close()
		}
	})
}

func (c *Client) serverAddrStr() string {
	return net.JoinHostPort(c.config.ServerAddr, strconv.Itoa(c.config.ServerPort))
}

func (c *Client) connect() error {

	conn, err := c.dialServer()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := utils.WriteMsg(conn, utils.MsgLogin, utils.LoginMsg{
		ClientID: c.config.ClientID,
		Token:    c.config.Token,
	}); err != nil {
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

	if err := c.applyConfig(conn); err != nil {
		_ = conn.Close()
		return err
	}

	c.mu.Lock()
	c.ctrlCon = conn
	c.runID = loginResp.RunID
	c.mu.Unlock()

	c.logger.Infof("Login successful, clientID=%s, runID=%s", c.config.ClientID, loginResp.RunID)
	go c.controlLoop(conn)
	return nil

}

func (c *Client) applyConfig(conn net.Conn) error {

	proxies := make([]utils.ApplyConfigProxyMsg, 0, len(c.config.Proxies))
	for _, proxy := range c.config.Proxies {
		proxies = append(proxies, utils.ApplyConfigProxyMsg{
			Name:       proxy.Name,
			Type:       proxy.Type,
			RemotePort: proxy.RemotePort,
		})
	}

	if err := utils.WriteMsg(conn, utils.MsgApplyConfig, utils.ApplyConfigMsg{Proxies: proxies}); err != nil {
		return fmt.Errorf("failed to send ApplyConfigMsg: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("failed to read ApplyConfigResp: %w", err)
	}
	if msgType != utils.MsgApplyConfigResp {
		return fmt.Errorf("expected ApplyConfigResp, got 0x%02x", msgType)
	}

	var resp utils.ApplyConfigRespMsg
	if err := utils.Decode(payload, &resp); err != nil {
		return fmt.Errorf("failed to parse ApplyConfigResp: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("apply config rejected by server: %s", resp.Error)
	}

	for _, proxy := range c.config.Proxies {
		c.logger.Infof("Proxy applied successfully: name=%s, remotePort=%d -> %s:%d", proxy.Name, proxy.RemotePort, proxy.LocalIP, proxy.LocalPort)
	}

	return nil

}

func (c *Client) dialServer() (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addr := c.serverAddrStr()
	if !c.config.TLS.Enabled {
		return dialer.Dial("tcp", addr)
	}
	config, err := c.tlsConfig()
	if err != nil {
		return nil, err
	}
	return tls.DialWithDialer(dialer, "tcp", addr, config)
}

func (c *Client) tlsConfig() (*tls.Config, error) {

	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.config.TLS.InsecureSkipVerify,
	}
	if c.config.TLS.ServerName != "" {
		config.ServerName = c.config.TLS.ServerName
	}
	if c.config.TLS.CAFile == "" {
		if err := c.loadClientCertificate(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	caCert, err := os.ReadFile(c.config.TLS.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tls ca_file: %w", err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append tls ca_file to cert pool")
	}
	config.RootCAs = pool
	if err := c.loadClientCertificate(config); err != nil {
		return nil, err
	}
	return config, nil

}

func (c *Client) loadClientCertificate(config *tls.Config) error {
	if c.config.TLS.CertFile == "" || c.config.TLS.KeyFile == "" {
		return fmt.Errorf("tls cert_file and key_file are required when tls is enabled")
	}
	cert, err := tls.LoadX509KeyPair(c.config.TLS.CertFile, c.config.TLS.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client tls certificate: %w", err)
	}
	config.Certificates = []tls.Certificate{cert}
	return nil
}

func (c *Client) releaseControlConn(conn net.Conn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctrlCon != conn {
		return false
	}
	c.ctrlCon = nil
	c.runID = ""
	return true
}

func (c *Client) handleDisconnect(conn net.Conn, reason string) {
	if !c.releaseControlConn(conn) {
		return
	}
	_ = conn.Close()
	c.logger.Warnf("Control connection lost: %s", reason)
	if c.retiring.Load() {
		return
	}
	c.tryReconnect()
}

func (c *Client) controlLoop(conn net.Conn) {

	heartbeatTicker := time.NewTicker(controlHeartbeatInterval)
	defer heartbeatTicker.Stop()

	msgCh := make(chan msgChan, 1)
	doneCh := make(chan struct{})
	defer close(doneCh)

	go func() {
		for {
			if err := conn.SetReadDeadline(time.Now().Add(controlReadTimeout)); err != nil {
				select {
				case msgCh <- msgChan{err: fmt.Errorf("failed to set read deadline: %w", err)}:
				case <-doneCh:
				}
				return
			}
			msgType, payload, err := utils.ReadMsg(conn)
			select {
			case msgCh <- msgChan{msgType, payload, err}:
			case <-doneCh:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-c.stopCh:
			_ = conn.Close()
			return
		case <-heartbeatTicker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(controlWriteTimeout)); err != nil {
				c.handleDisconnect(conn, fmt.Sprintf("failed to set write deadline: %v", err))
				return
			}
			err := utils.WriteMsg(conn, utils.MsgPing, utils.PingMsg{})
			_ = conn.SetWriteDeadline(time.Time{})
			if err != nil {
				c.handleDisconnect(conn, fmt.Sprintf("failed to send heartbeat: %v", err))
				return
			}
		case result := <-msgCh:
			if result.err != nil {
				select {
				case <-c.stopCh:
					return
				default:
				}
				c.handleDisconnect(conn, result.err.Error())
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

	workConn, err := c.dialServer()
	if err != nil {
		c.logger.Errorf("Failed to establish work connection [%s]: %v", msg.ProxyName, err)
		return
	}

	if err := utils.WriteMsg(workConn, utils.MsgStartWorkConn, utils.StartWorkConnMsg{WorkID: msg.WorkID}); err != nil {
		c.logger.Errorf("Failed to send StartWorkConn: %v", err)
		_ = workConn.Close()
		return
	}

	localAddr := net.JoinHostPort(proxy.LocalIP, strconv.Itoa(proxy.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		c.logger.Errorf("Failed to connect to local service [%s -> %s]: %v", msg.ProxyName, localAddr, err)
		_ = workConn.Close()
		return
	}

	c.logger.Debugf("Work connection bridged: proxy=%s, workID=%s, local=%s", msg.ProxyName, msg.WorkID, localAddr)

	utils.Pipe(workConn, localConn)

}

func (c *Client) findProxy(name string) *configs.ProxyConfig {
	for i := range c.config.Proxies {
		if c.config.Proxies[i].Name == name {
			return &c.config.Proxies[i]
		}
	}
	return nil
}

func (c *Client) tryReconnect() {
	backoff := reconnectInitialBackoff
	for {
		select {
		case <-c.stopCh:
			return
		case <-time.After(backoff):
			if c.retiring.Load() {
				return
			}
			c.logger.Infof("Reconnecting to server %s ...", c.serverAddrStr())
			if err := c.connect(); err != nil {
				c.logger.Errorf("Reconnect failed: %v, will retry in %v", err, backoff)
				backoff *= 2
				if backoff > reconnectMaxBackoff {
					backoff = reconnectMaxBackoff
				}
				continue
			}
			c.logger.Infof("Reconnect successful")
			return
		}
	}
}
