package internal

import (
	"crypto/tls"
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
	logger        *zap.Logger
	geoIP         *clients.GeoIP
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
	logger.Info("initialize database successfully")

	geoIP, err := clients.NewGeoIP(config.GeoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize geoip: %v", err)
	}
	logger.Info("GeoIP database loaded: " + config.GeoIP.DbPath)

	app := App{
		logger:        logger,
		tlsService:    services.NewTls(config.Cert, logger),
		serverService: services.NewServer(config.Server, logger, db, geoIP),
		geoIP:         geoIP,
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
		if a.geoIP != nil {
			_ = a.geoIP.Close()
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
		var ctrlWriteMu sync.Mutex
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
				if err := a.serverService.ProxiesApply(conn, &ctrlWriteMu, payload_, clientIdP, a.stopCh, clientStopCh); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to apply proxies: %v", err))
					return
				}
			case utils.MsgHeartbeat:
				if err := services.WriteCtrlMsg(&ctrlWriteMu, conn, utils.MsgHeartbeatResp, utils.HeartbeatRespMsg{}); err != nil {
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
