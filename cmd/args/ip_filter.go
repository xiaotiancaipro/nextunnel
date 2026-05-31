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

var ipFilterRules = []ipFilterRule{
	&ruleGeoIP{flagName: "ip-filter-allow-ip", status: 1, field: "ip"},
	&ruleGeoIP{flagName: "ip-filter-block-ip", status: 0, field: "ip"},
	&ruleGeoIP{flagName: "ip-filter-allow-country", status: 1, field: "country"},
	&ruleGeoIP{flagName: "ip-filter-block-country", status: 0, field: "country"},
	&ruleGeoIP{flagName: "ip-filter-allow-region", status: 1, field: "region"},
	&ruleGeoIP{flagName: "ip-filter-block-region", status: 0, field: "region"},
	&ruleGeoIP{flagName: "ip-filter-allow-city", status: 1, field: "city"},
	&ruleGeoIP{flagName: "ip-filter-block-city", status: 0, field: "city"},
	&ruleGlobal{flagName: "ip-filter-block-all", status: 0, category: "ALL"},
	&ruleGlobal{flagName: "ip-filter-allow-all", status: 1, category: "ALL"},
	&ruleGlobal{flagName: "ip-filter-block-local", status: 0, category: "LOCAL"},
	&ruleGlobal{flagName: "ip-filter-allow-local", status: 1, category: "LOCAL"},
	&ruleGlobal{flagName: "ip-filter-block-remote", status: 0, category: "REMOTE"},
	&ruleGlobal{flagName: "ip-filter-allow-remote", status: 1, category: "REMOTE"},
}

type ruleGeoIP struct {
	flagName string
	status   int16
	field    string
}

type ruleGlobal struct {
	flagName string
	status   int16
	category string
}

type ipFilterRule interface {
	run(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error)
}

func RunIPFilters(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {
	for i := range ipFilterRules {
		ran, err = ipFilterRules[i].run(cmd, cfg)
		if err != nil || ran {
			return ran, err
		}
	}
	return false, nil
}

func (g *ruleGeoIP) run(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed(g.flagName) {
		return false, nil
	}

	raw, err := cmd.Flags().GetString(g.flagName)
	if err != nil {
		return false, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}

	if g.field == "ip" {
		ip, err := utils.NormalizeIP(raw)
		if err != nil {
			return true, err
		}
		raw = *ip
	}

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return true, err
	}

	target, err := service.NewRuleTarget(g.field, raw)
	if err != nil {
		return true, err
	}

	return upsertAndPrint(cmd, service, target, g.status, "%s %s %s", ruleAction(g.status), g.field, raw)

}

func (c *ruleGlobal) run(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed(c.flagName) {
		return false, nil
	}

	enabled, err := cmd.Flags().GetBool(c.flagName)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return true, err
	}

	target, err := service.NewCategoryRuleTarget(c.category)
	if err != nil {
		return true, err
	}

	return upsertAndPrint(cmd, service, target, c.status, "%s category %s", ruleAction(c.status), c.category)

}

func newAccessRuleService(cfg *configs.Configs) (*services.AccessRule, error) {
	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return services.NewAccessRule(db), nil
}

func upsertAndPrint(cmd *cobra.Command, service *services.AccessRule, target services.RuleTarget, status int16, format string, args ...any) (bool, error) {
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
