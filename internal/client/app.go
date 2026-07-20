package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/apps"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/client/services"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"go.uber.org/zap"
)

type App struct {
	Configs  *configs.Configs
	logger   *zap.Logger
	services *services.Services
	apps     *apps.Apps
	stopOnce sync.Once
}

func (a *App) Init() error {
	logger, err := sharedlogger.NewLogger(a.Configs.Logs)
	if err != nil {
		return fmt.Errorf("failed to initialize logging: %v", err)
	}
	a.logger = logger
	a.initServices()
	if err := a.initApps(); err != nil {
		return err
	}
	return nil
}

func (a *App) Start() error {
	return a.apps.Conn.Start()
}

func (a *App) Stop() {
	a.stopOnce.Do(func() {
		if a.apps != nil {
			a.stopApps()
		}
		if a.logger != nil {
			a.logger.Info("Shutting down gracefully")
		}
	})
}

func (a *App) initServices() {
	tls := services.Tls{
		Config: a.Configs.Cert,
		Logger: a.logger,
	}
	server := services.Server{
		Config: a.Configs.Server,
	}
	client := services.Client{
		Config:  a.Configs.Client,
		Proxies: a.Configs.Proxies,
		Logger:  a.logger,
	}
	a.services = &services.Services{
		Client: &client,
		Server: &server,
		Tls:    &tls,
	}
}

func (a *App) initApps() error {
	conn := apps.Conn{
		Logger:   a.logger,
		Services: a.services,
	}
	if err := conn.Init(); err != nil {
		return fmt.Errorf("initialize Conn APP error: %w", err)
	}
	a.apps = &apps.Apps{
		Conn: &conn,
	}
	return nil
}

func (a *App) stopApps() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if a.apps.Conn != nil {
		_ = a.apps.Conn.Stop(ctx)
	}
}
