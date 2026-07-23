package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
	sharedlogger "github.com/xiaotiancaipro/nextunnel/internal/shared/logger"
)

const (
	ServerDefaultConfigPath = "nextunnel-server.toml"
	ServerEnvConfigPath     = "NEXTUNNEL_SERVER_CONFIG"
)

func LoadServerConfig(cmd *cobra.Command) (*configs.Configs, error) {
	spec := sharedcli.ConfigSpec{
		DefaultPath: ServerDefaultConfigPath,
		EnvVar:      ServerEnvConfigPath,
	}
	c, err := sharedcli.LoadConfig(cmd, spec, configs.Configs{})
	if err != nil {
		return nil, err
	}
	checks := []func() error{
		c.CheckCert,
		c.CheckDatabase,
		c.CheckIPLocation,
		c.CheckLogs,
		c.CheckServer,
		c.CheckServerWeb,
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

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

func CloseDatabase(db *clients.Database) {
	if db != nil {
		_ = db.Close()
	}
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
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &database, nil

}
