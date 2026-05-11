package internal

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-client/internal/services"
	"go.uber.org/zap"
)

type App struct {
	logger        *zap.Logger
	tlsService    *services.Tls
	serverService *services.Server
	clientService *services.Client
}

func NewApp(config *configs.Configs, logger *zap.Logger) *App {
	return &App{
		logger:        logger,
		tlsService:    services.NewTls(config, logger),
		serverService: services.NewServer(config, logger),
		clientService: services.NewClient(config, logger),
	}
}

func (a *App) Start() error {

	c, err := a.tlsService.Init()
	if err != nil {
		return err
	}

	conn, err := a.serverService.DialServer(c)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to connect to server: %s", err))
		return fmt.Errorf("failed to connect to server")
	}

	if err = a.clientService.Login(conn); err != nil {
		_ = conn.Close()
		a.logger.Error(fmt.Sprintf("Failed to login: %s", err))
		return fmt.Errorf("failed to login")
	}

	// TODO

}
