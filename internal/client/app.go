package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	services2 "github.com/xiaotiancaipro/nextunnel/internal/client/services"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type App struct {
	logger        *zap.Logger
	stopCh        chan struct{}
	stopOnce      sync.Once
	ctrlMu        sync.Mutex
	ctrlConn      net.Conn
	tlsService    *services2.Tls
	serverService *services2.Server
	clientService *services2.Client
}

type msgChan struct {
	msgType byte
	payload []byte
	err     error
}

func NewApp(config *configs.Configs) (*App, error) {

	logger, err := logger_.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	tls := services2.Tls{
		Config: config.Cert,
		Logger: logger,
	}
	server := services2.Server{
		Config: config.Server,
		Logger: logger,
	}
	client := services2.Client{
		Config:  config.Client,
		Proxies: config.Proxies,
		Logger:  logger,
	}

	app := App{
		logger:        logger,
		stopCh:        make(chan struct{}),
		tlsService:    &tls,
		serverService: &server,
		clientService: &client,
	}

	return &app, nil

}

func (a *App) Start() error {

	tlsCfg, err := a.tlsService.Init()
	if err != nil {
		a.logger.Error(fmt.Sprintf("failed to initialize TLS: %v", err))
		return err
	}

	a.clientService.DialWork = func() (net.Conn, error) {
		return a.serverService.DialServer(tlsCfg)
	}

	const (
		reconnectMinDelay = 2 * time.Second
		reconnectMaxDelay = 30 * time.Second
	)
	nextRetryDelay := reconnectMinDelay

	for {

		select {
		case <-a.stopCh:
			return nil
		default:
		}

		stopped := a.runSession(&nextRetryDelay, reconnectMinDelay)
		if stopped {
			return nil
		}

		select {
		case <-a.stopCh:
			return nil
		case <-time.After(nextRetryDelay):
		}

		if grow := nextRetryDelay * 2; grow > reconnectMaxDelay {
			nextRetryDelay = reconnectMaxDelay
		} else {
			nextRetryDelay = grow
		}

	}

}

func (a *App) Stop() {
	a.stopOnce.Do(func() {
		close(a.stopCh)
		a.ctrlMu.Lock()
		c := a.ctrlConn
		a.ctrlConn = nil
		a.ctrlMu.Unlock()
		if c != nil {
			_ = c.Close()
		}
		a.logger.Info("Shutting down gracefully")
	})
}

func (a *App) runSession(nextRetryDelay *time.Duration, reconnectMin time.Duration) (stopped bool) {

	conn, err := a.clientService.DialWork()
	if err != nil {
		a.logger.Warn(fmt.Sprintf("connect to server failed: %v", err))
		return false
	}

	a.ctrlMu.Lock()
	a.ctrlConn = conn
	a.ctrlMu.Unlock()
	defer func() {
		a.ctrlMu.Lock()
		if a.ctrlConn == conn {
			a.ctrlConn = nil
		}
		a.ctrlMu.Unlock()
		_ = conn.Close()
	}()

	a.clientService.Conn = conn
	a.logger.Info("Successfully connected to the server")

	runIdP, err := a.clientLogin()
	if err != nil {
		a.logger.Warn(fmt.Sprintf("login failed: %v", err))
		return false
	}
	a.logger.Info(fmt.Sprintf("Running with id: %s", *runIdP))

	if err = a.clientProxiesApply(); err != nil {
		a.logger.Warn(fmt.Sprintf("proxy registration failed: %v", err))
		return false
	}
	a.logger.Info("Client proxies configuration application successful")

	*nextRetryDelay = reconnectMin

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	msgCh := make(chan msgChan, 1)
	doneCh := make(chan struct{})
	defer close(doneCh)

	go a.controlLoop(conn, msgCh, doneCh, a.stopCh)

	for {
		select {
		case <-a.stopCh:
			return true
		case <-heartbeatTicker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				a.logger.Warn(fmt.Sprintf("heartbeat: set write deadline failed: %v", err))
				return false
			}
			err := protocol.WriteMsg(conn, protocol.MsgHeartbeat, protocol.HeartbeatMsg{})
			_ = conn.SetWriteDeadline(time.Time{})
			if err != nil {
				a.logger.Warn(fmt.Sprintf("heartbeat send failed: %v", err))
				return false
			}
		case result := <-msgCh:
			if result.err != nil {
				select {
				case <-a.stopCh:
					return true
				default:
				}
				a.logger.Warn(fmt.Sprintf("control read failed: %v", result.err))
				return false
			}
			switch result.msgType {
			case protocol.MsgNewWorkConn:
				var msg protocol.NewWorkConnMsg
				if err := protocol.Decode(result.payload, &msg); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to parse NewWorkConnMsg: %v", err))
					continue
				}
				go a.clientService.WorkConn(msg)
			case protocol.MsgHeartbeatResp:
			default:
				a.logger.Warn(fmt.Sprintf("Received unknown control message 0x%02x", result.msgType))
			}
		}
	}

}

func (a *App) clientLogin() (*string, error) {

	if err := a.clientService.Login(); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to login: %s", err))
		return nil, fmt.Errorf("failed to login")
	}

	runIdP, err := a.clientService.LoginResponse()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to login: %s", err))
		return nil, fmt.Errorf("failed to login")
	}

	return runIdP, nil

}

func (a *App) clientProxiesApply() error {
	if err := a.clientService.ProxiesApply(); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to apply proxies: %s", err))
		return fmt.Errorf("failed to apply proxies")
	}
	if err := a.clientService.ProxiesApplyResponse(); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to apply proxies: %s", err))
		return fmt.Errorf("failed to apply proxies")
	}
	return nil
}

func (a *App) controlLoop(conn net.Conn, msgCh chan msgChan, doneCh chan struct{}, stopNotify <-chan struct{}) {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
			select {
			case msgCh <- msgChan{err: fmt.Errorf("failed to set read deadline: %w", err)}:
			case <-stopNotify:
			case <-doneCh:
			}
			return
		}
		msgType, payload, err := protocol.ReadMsg(conn)
		select {
		case msgCh <- msgChan{msgType, payload, err}:
		case <-stopNotify:
			return
		case <-doneCh:
			return
		}
		if err != nil {
			return
		}
	}
}
