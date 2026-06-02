package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
)

var ipFilterRules = []ipFilterRule{
	&ipFilter{flag: "ip-filter-allow-ip", flagDel: "ip-filter-allow-ip-delete", status: 1, field: "ip"},
	&ipFilter{flag: "ip-filter-block-ip", flagDel: "ip-filter-block-ip-delete", status: 0, field: "ip"},
	&ipFilter{flag: "ip-filter-allow-country", flagDel: "ip-filter-allow-country-delete", status: 1, field: "country"},
	&ipFilter{flag: "ip-filter-block-country", flagDel: "ip-filter-block-country-delete", status: 0, field: "country"},
	&ipFilter{flag: "ip-filter-allow-region", flagDel: "ip-filter-allow-region-delete", status: 1, field: "region"},
	&ipFilter{flag: "ip-filter-block-region", flagDel: "ip-filter-block-region-delete", status: 0, field: "region"},
	&ipFilter{flag: "ip-filter-allow-city", flagDel: "ip-filter-allow-city-delete", status: 1, field: "city"},
	&ipFilter{flag: "ip-filter-block-city", flagDel: "ip-filter-block-city-delete", status: 0, field: "city"},
	&ipFilter{flag: "ip-filter-allow-all", flagDel: "ip-filter-allow-all-delete", status: 1, field: "ALL"},
	&ipFilter{flag: "ip-filter-block-all", flagDel: "ip-filter-block-all-delete", status: 0, field: "ALL"},
	&ipFilter{flag: "ip-filter-allow-local", flagDel: "ip-filter-allow-local-delete", status: 1, field: "LOCAL"},
	&ipFilter{flag: "ip-filter-block-local", flagDel: "ip-filter-block-local-delete", status: 0, field: "LOCAL"},
	&ipFilter{flag: "ip-filter-allow-remote", flagDel: "ip-filter-allow-remote-delete", status: 1, field: "REMOTE"},
	&ipFilter{flag: "ip-filter-block-remote", flagDel: "ip-filter-block-remote-delete", status: 0, field: "REMOTE"},
}

type ipFilterRule interface {
	run(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error)
}

type ipFilter struct {
	flag    string
	flagDel string
	status  int16
	field   string
}

func RunIPFilterList(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if !cmd.Flags().Changed("ip-filter-list") {
		return false, nil
	}

	enabled, err := cmd.Flags().GetBool("ip-filter-list")
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

	rules, err := service.ListRules()
	if err != nil {
		return true, err
	}
	if len(rules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ip filter rules")
		return true, nil
	}

	for i := range rules {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), formatAccessRule(rules[i]))
	}

	return true, nil

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

func (f *ipFilter) run(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	if cmd.Flags().Changed(f.flagDel) {
		return f.runDelete(cmd, cfg)
	}

	if !cmd.Flags().Changed(f.flag) {
		return false, nil
	}

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return true, err
	}

	if f.isCategory() {
		enabled, err := cmd.Flags().GetBool(f.flag)
		if err != nil {
			return false, err
		}
		if !enabled {
			return false, nil
		}
		target, err := service.NewCategoryRuleTarget(f.field)
		if err != nil {
			return true, err
		}
		return f.upsertAndPrint(cmd, service, target, f.status, "%s category %s", f.ruleAction(f.status), f.field)
	}

	raw, err := cmd.Flags().GetString(f.flag)
	if err != nil {
		return false, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}

	if f.field == "ip" {
		ip, err := utils.NormalizeIP(raw)
		if err != nil {
			return true, err
		}
		raw = *ip
	}

	target, err := service.NewRuleTarget(f.field, raw)
	if err != nil {
		return true, err
	}

	return f.upsertAndPrint(cmd, service, target, f.status, "%s %s %s", f.ruleAction(f.status), f.field, raw)

}

func (f *ipFilter) runDelete(cmd *cobra.Command, cfg *configs.Configs) (ran bool, err error) {

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return true, err
	}

	if f.isCategory() {
		enabled, err := cmd.Flags().GetBool(f.flagDel)
		if err != nil {
			return false, err
		}
		if !enabled {
			return false, nil
		}
		target, err := service.NewCategoryRuleTarget(f.field)
		if err != nil {
			return true, err
		}
		return f.deleteAndPrint(cmd, service, target, f.status, "deleted %s category %s", f.ruleAction(f.status), f.field)
	}

	raw, err := cmd.Flags().GetString(f.flagDel)
	if err != nil {
		return false, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}

	if f.field == "ip" {
		ip, err := utils.NormalizeIP(raw)
		if err != nil {
			return true, err
		}
		raw = *ip
	}

	target, err := service.NewRuleTarget(f.field, raw)
	if err != nil {
		return true, err
	}

	return f.deleteAndPrint(cmd, service, target, f.status, "deleted %s %s %s", f.ruleAction(f.status), f.field, raw)

}

func (f *ipFilter) isCategory() bool {
	switch f.field {
	case "ALL", "LOCAL", "REMOTE":
		return true
	default:
		return false
	}
}

func (f *ipFilter) upsertAndPrint(cmd *cobra.Command, service *services.AccessRule, target services.RuleTarget, status int16, format string, args ...any) (bool, error) {
	if err := service.UpsertRule(target, status); err != nil {
		return true, err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
	return true, nil
}

func (f *ipFilter) deleteAndPrint(cmd *cobra.Command, service *services.AccessRule, target services.RuleTarget, status int16, format string, args ...any) (bool, error) {
	if err := service.DeleteRule(target, status); err != nil {
		return true, err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
	return true, nil
}

func (f *ipFilter) ruleAction(status int16) string {
	if status == 1 {
		return "Allowed"
	}
	return "Blocked"
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

func formatAccessRule(rule models.AccessRule) string {
	action := "Blocked"
	if rule.Status == 1 {
		action = "Allowed"
	}
	switch {
	case rule.Category != nil:
		return fmt.Sprintf("%s category %s", action, *rule.Category)
	case rule.Ip != nil:
		return fmt.Sprintf("%s ip %s", action, *rule.Ip)
	case rule.Country != nil:
		return fmt.Sprintf("%s country %s", action, *rule.Country)
	case rule.Region != nil:
		return fmt.Sprintf("%s region %s", action, *rule.Region)
	case rule.City != nil:
		return fmt.Sprintf("%s city %s", action, *rule.City)
	default:
		return action
	}
}
