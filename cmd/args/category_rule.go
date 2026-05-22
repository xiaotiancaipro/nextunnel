package args

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
)

type CategoryRule struct {
	FlagName string
	Status   int16
	Category string
}

func (c *CategoryRule) New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed(c.FlagName) {
		return false, nil
	}

	enabled, err := cmd.Flags().GetBool(c.FlagName)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return true, fmt.Errorf("failed to initialize logging: %w", err)
	}

	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return true, fmt.Errorf("failed to initialize database: %w", err)
	}

	service := services.RulesIp{DB: db}
	target, err := service.NewCategoryRuleTarget(c.Category)
	if err != nil {
		return true, err
	}
	if err := service.UpsertRule(target, c.Status); err != nil {
		return true, err
	}

	action := "blocked"
	if c.Status == 1 {
		action = "allowed"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s category %s\n", action, c.Category)
	return true, nil

}
