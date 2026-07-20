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

func NewClientRegistryFromConfig(cfg *configs.Configs) (*services.Client, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &services.Client{DB: db}, nil
}

func NewAccessRuleFromConfig(cfg *configs.Configs) (*services.AccessRule, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &services.AccessRule{DB: db}, nil
}

func NewClientRegistryAndCertFromConfig(cfg *configs.Configs) (*services.Client, *services.ClientCert, error) {
	db, err := NewDBFromConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return &services.Client{DB: db}, &services.ClientCert{DB: db, CertDir: cfg.Cert.Dir, ListenHost: cfg.Cert.Host}, nil
}
