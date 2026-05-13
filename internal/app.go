package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
)

type App struct {
	logger        *zap.Logger
	stopCh        chan struct{}
	tlsService    *services.Tls
	serverService *services.Server
}

func NewApp(config *configs.Configs) (*App, error) {

	logger, err := utils.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	tlsService := services.Tls{
		Config: config.Tls,
		Logger: logger,
	}
	serverService := services.Server{
		Config:     config.Server,
		IpBlackMap: utils.SetupLookupMap(config.Server.IpBlacklist),
		Logger:     logger,
	}

	app := App{
		logger:        logger,
		stopCh:        make(chan struct{}),
		tlsService:    &tlsService,
		serverService: &serverService,
	}

	return &app, nil

}

func (a *App) Start() error {

	listener, err := a.serverService.Listen()
	if err != nil {
		return err
	}
	a.logger.Info("Listening on " + listener.Addr().String())

	tlsConfig, err := a.tlsService.Init()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to initialize TLS connection: %v", err))
		return err
	}
	a.logger.Info("TLS connection established")

	for {
		connRaw, err := listener.Accept()
		if err != nil {
			select {
			case <-a.stopCh:
				return nil
			default:
			}
			a.logger.Error(fmt.Sprintf("Failed to accept connection: %v", err))
			return nil
		}
		conn, err := a.serverService.EstablishConn(connRaw, tlsConfig)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to incoming TLS connection: %v", err))
			_ = connRaw.Close()
			continue
		}
		go a.acceptedConn(conn)
	}

}

func (a *App) acceptedConn(conn net.Conn) {

	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to read first message [%s]: %v", conn.RemoteAddr(), err))
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case utils.MsgLogin:
		clientIdP, runIdP, err := a.serverService.Login(conn, payload)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to login: %v", err))
			return
		}
		for {
			msgType_, payload_, err := utils.ReadMsg(conn)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Client control connection disconnected, clientID=%s, runID=%s: %v", *clientIdP, *runIdP, err))
				return
			}
			switch msgType_ {
			case utils.MsgProxiesApply:
				if err := a.serverService.ProxiesApply(conn, payload_, clientIdP, a.stopCh); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to apply proxies: %v", err))
					return
				}
			case utils.MsgHeartbeat:
				if err := utils.WriteMsg(conn, utils.MsgHeartbeatResp, utils.HeartbeatRespMsg{}); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to send HeartbeatRespMsg: %v", err))
					return
				}
			default:
				a.logger.Error(fmt.Sprintf("Unknown message received on control connection 0x%02x runID=%s", msgType_, *runIdP))
			}
		}
	case utils.MsgStartWorkConn:
		if err := a.serverService.StartWorkConn(payload); err != nil {
			a.logger.Error("Failed to start work connection")
			return
		}
	default:
		a.logger.Error(fmt.Sprintf("Unknown first message type 0x%02x [%s]", msgType, conn.RemoteAddr()))
	}

}
