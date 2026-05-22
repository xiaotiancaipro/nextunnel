package internal

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
	"go.uber.org/zap"
)

type App struct {
	configs       *configs.Configs
	logger        *zap.Logger
	tlsService    *services.Tls
	serverService *services.Server
	stopCh        chan struct{}
	stopOnce      sync.Once
	listenerMu    sync.Mutex
	listener      net.Listener
}

func NewApp(config *configs.Configs) (*App, error) {

	logger, err := logger_.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	db, err := clients.NewDB(config.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	tlsService := services.Tls{
		Config:     config.Tls,
		ServerAddr: config.Server.Addr,
		Logger:     logger,
	}
	serverService := services.Server{
		Config: config.Server,
		Logger: logger,
		DB:     db,
	}

	app := App{
		configs:       config,
		logger:        logger,
		tlsService:    &tlsService,
		serverService: &serverService,
		stopCh:        make(chan struct{}),
	}

	return &app, nil

}

func (a *App) Start() error {

	listener, err := a.serverService.Listen()
	if err != nil {
		return err
	}
	a.listenerMu.Lock()
	a.listener = listener
	a.listenerMu.Unlock()

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
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			a.logger.Error(fmt.Sprintf("Failed to accept connection: %v", err))
			return err
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

func (a *App) Stop() {
	a.stopOnce.Do(func() {
		close(a.stopCh)
		a.listenerMu.Lock()
		ln := a.listener
		a.listener = nil
		a.listenerMu.Unlock()
		if ln != nil {
			_ = ln.Close()
		}
		a.logger.Info("Shutting down gracefully")
	})
}

func (a *App) acceptedConn(conn net.Conn) {

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to read first message [%s]: %v", conn.RemoteAddr(), err))
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case utils.MsgLogin:
		defer func() { _ = conn.Close() }()
		clientIdP, runIdP, err := a.serverService.Login(conn, payload)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to login: %v", err))
			return
		}
		clientStopCh := make(chan struct{})
		defer close(clientStopCh)
		for {
			msgType_, payload_, err := utils.ReadMsg(conn)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Client control connection disconnected, clientID=%s, runID=%s: %v", *clientIdP, *runIdP, err))
				return
			}
			switch msgType_ {
			case utils.MsgProxiesApply:
				if err := a.serverService.ProxiesApply(conn, payload_, clientIdP, a.stopCh, clientStopCh); err != nil {
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
		if err := a.serverService.StartWorkConn(conn, payload); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to start work connection: %v", err))
			_ = conn.Close()
			return
		}
		return
	default:
		a.logger.Error(fmt.Sprintf("Unknown first message type 0x%02x [%s]", msgType, conn.RemoteAddr()))
		_ = conn.Close()
	}

}
