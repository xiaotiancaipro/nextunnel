package cli

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
)

func NewClientRegistryFromConfig(cfg *configs.Configs) (*services.Client, error) {
	database, err := newDatabaseFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &services.Client{Database: database}, nil
}

func NewAccessRuleFromConfig(cfg *configs.Configs) (*services.AccessRule, error) {
	database, err := newDatabaseFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &services.AccessRule{Database: database}, nil
}

func NewClientRegistryAndCertFromConfig(cfg *configs.Configs) (*services.Client, *services.ClientCert, error) {
	database, err := newDatabaseFromConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	client := services.Client{Database: database}
	clientCert := services.ClientCert{
		Config:   cfg.Cert,
		Database: database,
	}
	return &client, &clientCert, nil
}

func newDatabaseFromConfig(configs *configs.Configs) (*clients.Database, error) {

	logger, err := sharedlogger.NewLogger(configs.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	database := clients.Database{
		Config: configs.Database,
		Logger: logger,
	}
	if err := database.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return &database, nil

}
