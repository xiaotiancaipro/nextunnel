package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
)

type GeoRule struct {
	FlagName string
	Status   int16
	Field    string
}

func (g *GeoRule) New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed(g.FlagName) {
		return false, nil
	}

	raw, err := cmd.Flags().GetString(g.FlagName)
	if err != nil {
		return false, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
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

	target, err := service.NewRuleTarget(g.Field, raw)
	if err != nil {
		return true, err
	}

	if err := service.UpsertRule(target, g.Status); err != nil {
		return true, err
	}

	action := "blocked"
	if g.Status == 1 {
		action = "allowed"
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", action, g.Field, raw)
	return true, nil

}
