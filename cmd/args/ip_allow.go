package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
)

type IpAllow struct{}

func (*IpAllow) New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed("ip-allow") {
		return false, nil
	}

	raw, err := cmd.Flags().GetString("ip-allow")
	if err != nil {
		return false, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}

	ip, err := utils.NormalizeIP(raw)
	if err != nil {
		return true, err
	}

	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return true, fmt.Errorf("failed to initialize logging: %w", err)
	}

	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return true, fmt.Errorf("failed to initialize database: %w", err)
	}

	service := services.IpAddress{
		DB: db,
	}
	if err := service.UpsertIPStatus(*ip, 1); err != nil {
		return true, err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "allowed ip %s\n", *ip)
	return true, nil

}
