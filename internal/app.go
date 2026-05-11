package internal

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-client/internal/services"
	"go.uber.org/zap"
)

type App struct {
	logger        *zap.Logger
	serverService *services.Server
	tlsService    *services.Tls
}

func NewApp(config *configs.Configs, logger *zap.Logger) *App {
	return &App{
		serverService: services.NewServer(config, logger),
		tlsService:    services.NewTls(config, logger),
	}
}

func (a *App) Start() error {

	conn, err := a.dialServer()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to connect to server: %s", err))
		return fmt.Errorf("failed to connect to server")
	}

}

func (a *App) dialServer() (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addr := a.serverService.AddrStr()
	c, err := a.tlsService.Init()
	if err != nil {
		return nil, err
	}
	return tls.DialWithDialer(dialer, "tcp", addr, c)
}
