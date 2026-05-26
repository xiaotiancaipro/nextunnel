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

var IPFilterRules = []IpFilterRule{
	&RuleGeoIP{FlagName: "ip-filter-allow-ip", Status: 1, Field: "ip"},
	&RuleGeoIP{FlagName: "ip-filter-block-ip", Status: 0, Field: "ip"},
	&RuleGeoIP{FlagName: "ip-filter-allow-country", Status: 1, Field: "country"},
	&RuleGeoIP{FlagName: "ip-filter-block-country", Status: 0, Field: "country"},
	&RuleGeoIP{FlagName: "ip-filter-allow-region", Status: 1, Field: "region"},
	&RuleGeoIP{FlagName: "ip-filter-block-region", Status: 0, Field: "region"},
	&RuleGeoIP{FlagName: "ip-filter-allow-city", Status: 1, Field: "city"},
	&RuleGeoIP{FlagName: "ip-filter-block-city", Status: 0, Field: "city"},
	&RuleGlobal{FlagName: "ip-filter-block-all", Status: 0, Category: "ALL"},
	&RuleGlobal{FlagName: "ip-filter-allow-all", Status: 1, Category: "ALL"},
	&RuleGlobal{FlagName: "ip-filter-block-local", Status: 0, Category: "LOCAL"},
	&RuleGlobal{FlagName: "ip-filter-allow-local", Status: 1, Category: "LOCAL"},
	&RuleGlobal{FlagName: "ip-filter-block-remote", Status: 0, Category: "REMOTE"},
	&RuleGlobal{FlagName: "ip-filter-allow-remote", Status: 1, Category: "REMOTE"},
}

type RuleGeoIP struct {
	FlagName string
	Status   int16
	Field    string
}

type RuleGlobal struct {
	FlagName string
	Status   int16
	Category string
}

type IpFilterRule interface {
	New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error)
}

func (g *RuleGeoIP) New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

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

	if g.Field == "ip" {
		ip, err := utils.NormalizeIP(raw)
		if err != nil {
			return true, err
		}
		raw = *ip
	}

	service, err := newRulesIpService(cfg)
	if err != nil {
		return true, err
	}

	target, err := service.NewRuleTarget(g.Field, raw)
	if err != nil {
		return true, err
	}

	return upsertAndPrint(cmd, service, target, g.Status, "%s %s %s", ruleAction(g.Status), g.Field, raw)

}

func (c *RuleGlobal) New(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

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

	service, err := newRulesIpService(cfg)
	if err != nil {
		return true, err
	}

	target, err := service.NewCategoryRuleTarget(c.Category)
	if err != nil {
		return true, err
	}

	return upsertAndPrint(cmd, service, target, c.Status, "%s category %s", ruleAction(c.Status), c.Category)

}

func newRulesIpService(cfg *configs.Configs) (*services.RulesIp, error) {
	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return &services.RulesIp{DB: db}, nil
}

func upsertAndPrint(cmd *cobra.Command, service *services.RulesIp, target services.RuleTarget, status int16, format string, args ...any) (bool, error) {
	if err := service.UpsertRule(target, status); err != nil {
		return true, err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
	return true, nil
}

func ruleAction(status int16) string {
	if status == 1 {
		return "allowed"
	}
	return "blocked"
}
