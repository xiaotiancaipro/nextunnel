package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/apps"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"go.uber.org/zap"
)

type App struct {
	Configs  *configs.Configs
	logger   *zap.Logger
	clients  *clients.Clients
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
	a.logger.Info("server initializing")
	if err := a.initClients(); err != nil {
		return err
	}
	a.initServices()
	if err := a.initApps(); err != nil {
		return err
	}
	a.logger.Info("server initialized")
	return nil
}

func (a *App) Start() error {
	errors := make(chan error, 2)
	go func() {
		if err := a.apps.Web.Start(); err != nil {
			a.logger.Error(fmt.Sprintf("web server stopped: %v", err))
			errors <- err
		}
	}()
	go func() {
		if err := a.apps.Conn.Start(); err != nil {
			a.logger.Error(fmt.Sprintf("conn server stopped: %v", err))
			errors <- err
		}
	}()
	return <-errors
}

func (a *App) Stop() {
	a.stopOnce.Do(func() {
		if a.apps != nil {
			a.stopApps()
		}
		a.stopClients()
		if a.logger != nil {
			a.logger.Info("shutting down gracefully")
		}
	})
}

func (a *App) initClients() error {
	database := clients.Database{
		Config: a.Configs.Database,
		Logger: a.logger,
	}
	if err := database.Init(); err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	ipLocation := clients.IPLocation{
		Config: a.Configs.IPLocation,
		Logger: a.logger,
	}
	if err := ipLocation.Init(); err != nil {
		return fmt.Errorf("failed to initialize ip location: %v", err)
	}
	a.clients = &clients.Clients{
		Database:   &database,
		IPLocation: &ipLocation,
	}
	return nil
}

func (a *App) initServices() {
	client := services.Client{Database: a.clients.Database}
	clientCert := services.ClientCert{
		Database: a.clients.Database,
		Config:   a.Configs.Cert,
	}
	clientProxy := services.ClientProxy{Database: a.clients.Database}
	proxyBroker := services.ProxyBroker{Logger: a.logger}
	accessRule := services.AccessRule{Database: a.clients.Database}
	accessLog := services.AccessLog{
		Database:           a.clients.Database,
		ClientService:      &client,
		ClientProxyService: &clientProxy,
	}
	accessFilter := services.AccessFilter{
		Logger:            a.logger,
		Database:          a.clients.Database,
		IPLocation:        a.clients.IPLocation,
		AccessRuleService: &accessRule,
		AccessLogService:  &accessLog,
	}
	session := services.Session{
		Logger:              a.logger,
		Database:            a.clients.Database,
		ClientService:       &client,
		ClientProxyService:  &clientProxy,
		ProxyBrokerService:  &proxyBroker,
		AccessFilterService: &accessFilter,
	}
	tls := services.Tls{
		Config: a.Configs.Cert,
		Logger: a.logger,
	}
	a.services = &services.Services{
		AccessFilter: &accessFilter,
		AccessLog:    &accessLog,
		AccessRule:   &accessRule,
		Client:       &client,
		ClientCert:   &clientCert,
		ClientProxy:  &clientProxy,
		ProxyBroker:  &proxyBroker,
		Session:      &session,
		Tls:          &tls,
	}
}

func (a *App) initApps() error {
	web := apps.Web{
		Config:   a.Configs,
		Logger:   a.logger,
		Services: a.services,
	}
	if err := web.Init(); err != nil {
		return fmt.Errorf("initialize API APP error: %w", err)
	}
	conn := apps.Conn{
		Config:   a.Configs,
		Logger:   a.logger,
		Services: a.services,
	}
	_ = conn.Init()
	a.apps = &apps.Apps{
		Web:  &web,
		Conn: &conn,
	}
	return nil
}

func (a *App) stopClients() {
	if a.clients == nil {
		return
	}
	if a.clients.IPLocation != nil {
		_ = a.clients.IPLocation.Close()
	}
	if a.clients.Database != nil {
		_ = a.clients.Database.Close()
	}
}

func (a *App) stopApps() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if a.apps.Conn != nil {
		_ = a.apps.Conn.Stop(ctx)
	}
	if a.apps.Web != nil {
		_ = a.apps.Web.Stop(ctx)
	}
}
