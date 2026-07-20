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

// CloseDatabase closes the database connection. Safe for nil and repeated calls.
func CloseDatabase(db *clients.Database) {
	if db != nil {
		_ = db.Close()
	}
}

// ExitOnDBErr closes db before exiting, because sharedcli.ExitOnErr uses os.Exit
// and skips deferred Close.
func ExitOnDBErr(cmd *cobra.Command, err error, db *clients.Database) {
	if err != nil {
		CloseDatabase(db)
		sharedcli.ExitOnErr(cmd, err)
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
