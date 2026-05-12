package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-client/internal/services"
	"github.com/xiaotiancaipro/nextunnel-client/internal/utils"
	"go.uber.org/zap"
)

type App struct {
	logger        *zap.Logger
	stopCh        chan struct{}
	tlsService    *services.Tls
	serverService *services.Server
	clientService *services.Client
}

type msgChan struct {
	msgType byte
	payload []byte
	err     error
}

func NewApp(config *configs.Configs) (*App, error) {

	logger, err := utils.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	tls := services.Tls{
		Config: config.Tls,
		Logger: logger,
	}
	server := services.Server{
		Config: config.Server,
		Logger: logger,
	}
	client := services.Client{
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

	conn, err := a.serverConn()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	a.logger.Info("Successfully connected to the server")

	runIdP, err := a.clientLogin()
	if err != nil {
		return err
	}
	a.logger.Info(fmt.Sprintf("Running with id: %s", *runIdP))

	if err = a.clientProxiesApply(); err != nil {
		return err
	}
	a.logger.Info("Client proxies configuration application successful")

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	msgCh := make(chan msgChan, 1)
	doneCh := make(chan struct{})
	defer close(doneCh)

	go a.controlLoop(conn, msgCh, doneCh)

	for {
		select {
		case <-a.stopCh:
			return nil
		case <-heartbeatTicker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				a.logger.Error(fmt.Sprintf("failed to set write deadline: %v", err))
				return nil
			}
			err := utils.WriteMsg(conn, utils.MsgHeartbeat, utils.HeartbeatMsg{})
			_ = conn.SetWriteDeadline(time.Time{})
			if err != nil {
				a.logger.Error(fmt.Sprintf("failed to send heartbeat: %v", err))
				return nil
			}
		case result := <-msgCh:
			if result.err != nil {
				select {
				case <-a.stopCh:
					return nil
				default:
				}
				a.logger.Error(fmt.Sprintf("Error: %v", result.err.Error()))
				return nil
			}
			switch result.msgType {
			case utils.MsgNewWorkConn:
				var msg utils.NewWorkConnMsg
				if err := utils.Decode(result.payload, &msg); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to parse NewWorkConnMsg: %v", err))
					continue
				}
				go a.clientService.WorkConn(msg)
			case utils.MsgHeartbeatResp:
			default:
				a.logger.Warn(fmt.Sprintf("Received unknown control message 0x%02x", result.msgType))
			}
		}
	}

}

func (a *App) serverConn() (net.Conn, error) {

	c, err := a.tlsService.Init()
	if err != nil {
		return nil, err
	}

	conn, err := a.serverService.DialServer(c)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to connect to server: %s", err))
		return nil, fmt.Errorf("failed to connect to server")
	}

	a.clientService.Conn = conn
	return conn, nil

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

func (a *App) controlLoop(conn net.Conn, msgCh chan msgChan, doneCh chan struct{}) {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
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
}
