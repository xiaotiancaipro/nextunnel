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
		Config: config.Server,
		Logger: logger,
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
		// TODO
	case utils.MsgStartWorkConn:
		// TODO
	default:
		a.logger.Error(fmt.Sprintf("Unknown first message type 0x%02x [%s]", msgType, conn.RemoteAddr()))
		_ = conn.Close()
	}
}
