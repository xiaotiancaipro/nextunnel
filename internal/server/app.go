package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiaotiancaipro/nextunnel/internal/server/apps"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	apps       *apps.Apps
	configs    *configs.Configs
	logger     *zap.Logger
	db         *gorm.DB
	ipLocation *clients.IPLocation
	services   *services.Services
}

func (a *App) Init(config *configs.Configs) error {

	a.configs = config

	logger, err := sharedlogger.NewLogger(config.Logs)
	if err != nil {
		return fmt.Errorf("failed to initialize logging: %v", err)
	}
	a.logger = logger

	db, err := clients.NewDB(config.Database, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	logger.Info("initialize database successfully")
	a.db = db

	ipLocation, err := clients.NewIPLocation(config.IPLocation.APIKey, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize ip location: %v", err)
	}
	a.ipLocation = ipLocation

	a.initServices()

	if err := a.initApps(); err != nil {
		return err
	}

	return nil

}

func (a *App) initServices() {
	client := services.Client{DB: a.db}
	clientCert := services.ClientCert{
		DB:     a.db,
		Config: a.configs.Cert,
	}
	clientProxy := services.ClientProxy{DB: a.db}
	accessRule := services.AccessRule{DB: a.db}
	accessLog := services.AccessLog{
		DB:                 a.db,
		ClientService:      &client,
		ClientProxyService: &clientProxy,
	}
	server := services.Server{
		Config:             a.configs.Server,
		Logger:             a.logger,
		DB:                 a.db,
		IPLocation:         a.ipLocation,
		ClientService:      &client,
		ClientProxyService: &clientProxy,
		AccessRuleService:  &accessRule,
		AccessLogService:   &accessLog,
	}
	tls := services.Tls{
		Config: a.configs.Cert,
		Logger: a.logger,
	}
	a.services = &services.Services{
		AccessLog:   &accessLog,
		AccessRule:  &accessRule,
		Client:      &client,
		ClientCert:  &clientCert,
		ClientProxy: &clientProxy,
		Server:      &server,
		Tls:         &tls,
	}
}

func (a *App) initApps() error {

	var apiApp apps.API
	if a.configs.Web.IsEnabled() {
		apiApp = apps.API{
			Config:   a.configs,
			Logger:   a.logger,
			Services: a.services,
		}
		if err := apiApp.Init(); err != nil {
			return fmt.Errorf("initialize API APP error, " + err.Error())
		}
	}

	connApp := apps.Conn{
		Config:     a.configs,
		Logger:     a.logger,
		DB:         a.db,
		IPLocation: a.ipLocation,
		Services:   a.services,
	}

	a.apps = &apps.Apps{
		API:  &apiApp,
		Conn: &connApp,
	}
	return nil

}

func (a *App) Start() error {

	start := func() {
		go func() {
			if err := a.apps.API.Start(); err != nil {
				return
			}
		}()
		go func() {
			if err := a.apps.Conn.Start(); err != nil {
				return
			}
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		signal.Stop(sigCh)
		ExitOnErr(cmd, err)
	case <-sigCh:
		signal.Stop(sigCh)
		app.Stop()
		err := <-errCh
		ExitOnErr(cmd, err)
	}
}
