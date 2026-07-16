package cli

import (
	"fmt"

	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
	"gorm.io/gorm"
)

func NewDBFromConfig(cfg *configs.Configs) (*gorm.DB, error) {
	logger, err := sharedlogger.NewLogger(cfg.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

func NewClientRegistryFromConfig(cfg *configs.Configs) (*services.ClientRegistry, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return services.NewClientRegistry(db), nil
}

func NewAccessRuleFromConfig(cfg *configs.Configs) (*services.AccessRule, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return services.NewAccessRule(db), nil
}

func NewClientRegistryAndCertFromConfig(cfg *configs.Configs) (*services.ClientRegistry, *services.ClientCertRegistry, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return services.NewClientRegistry(db), services.NewClientCertRegistry(db, cfg.Cert.Dir, cfg.Cert.Host), nil
}
