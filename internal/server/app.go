package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	clients2 "github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	services2 "github.com/xiaotiancaipro/nextunnel/internal/server/services"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	version       string
	config        *configs.Configs
	logger        *zap.Logger
	db            *gorm.DB
	ipLocator     clients2.IPLocator
	tlsService    *services2.Tls
	serverService *services2.Server
	webServer     *controller.Server
	stopCh        chan struct{}
	stopOnce      sync.Once
	listenerMu    sync.Mutex
	listener      net.Listener
}

func NewApp(config *configs.Configs, version string) (*App, error) {

	logger, err := logger_.NewLogger(config.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %v", err)
	}

	db, err := clients2.NewDB(config.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}
	logger.Info("initialize database successfully")

	ipLocator, err := clients2.NewIPLocator(config.IPLocation, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ip location: %v", err)
	}
	switch config.IPLocation.Type {
	case "api":
		logger.Info("IP location provider: api")
	default:
		logger.Info("IP location provider: geoip, database=" + config.IPLocation.GeoIPDbPath)
	}

	app := App{
		version:       version,
		config:        config,
		logger:        logger,
		db:            db,
		tlsService:    services2.NewTls(config.Cert, logger),
		serverService: services2.NewServer(config.Server, logger, db, ipLocator),
		ipLocator:     ipLocator,
		stopCh:        make(chan struct{}),
	}

	if config.Web.IsEnabled() {
		app.webServer = controller.NewServer(version, config, db, logger)
	}

	return &app, nil

}

func (a *App) Start() error {

	if a.webServer != nil {
		go func() {
			if err := a.webServer.Start(); err != nil {
				a.logger.Error(fmt.Sprintf("Web management API stopped: %v", err))
			}
		}()
	}

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
		go a.handleConn(connRaw, tlsConfig)
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
		if a.webServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = a.webServer.Stop(ctx)
		}
		if a.ipLocator != nil {
			_ = a.ipLocator.Close()
		}
		a.logger.Info("Shutting down gracefully")
	})
}

func (a *App) handleConn(connRaw net.Conn, tlsConfig *tls.Config) {
	conn, err := a.serverService.EstablishConn(connRaw, tlsConfig)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to incoming TLS connection: %v", err))
		_ = connRaw.Close()
		return
	}
	a.acceptedConn(conn)
}

func (a *App) acceptedConn(conn net.Conn) {

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := protocol.ReadMsg(conn)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to read first message [%s]: %v", conn.RemoteAddr(), err))
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case protocol.MsgLogin:
		clientIdP, runIdP, err := a.serverService.Login(conn, payload)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to login: %v", err))
			_ = conn.Close()
			return
		}
		clientID := *clientIdP
		clientStopCh := make(chan struct{})
		defer func() {
			close(clientStopCh)
			if err := a.serverService.SetClientProxiesOffline(clientID); err != nil {
				a.logger.Warn(fmt.Sprintf("Failed to mark client proxies offline: clientID=%s, err=%v", clientID, err))
			}
			_ = conn.Close()
		}()
		var ctrlWriteMu sync.Mutex
		for {
			msgType_, payload_, err := protocol.ReadMsg(conn)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Client control connection disconnected, clientID=%s, runID=%s: %v", *clientIdP, *runIdP, err))
				return
			}
			switch msgType_ {
			case protocol.MsgProxiesApply:
				if err := a.serverService.ProxiesApply(conn, &ctrlWriteMu, payload_, clientIdP, a.stopCh, clientStopCh); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to apply proxies: %v", err))
					return
				}
			case protocol.MsgHeartbeat:
				if err := services2.WriteCtrlMsg(&ctrlWriteMu, conn, protocol.MsgHeartbeatResp, protocol.HeartbeatRespMsg{}); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to send HeartbeatRespMsg: %v", err))
					return
				}
			default:
				a.logger.Error(fmt.Sprintf("Unknown message received on control connection 0x%02x runID=%s", msgType_, *runIdP))
			}
		}
	case protocol.MsgStartWorkConn:
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
