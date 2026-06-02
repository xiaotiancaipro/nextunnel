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

func ListIPFilters(cmd *cobra.Command, cfg *configs.Configs) error {

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return err
	}

	rules, err := service.ListRules()
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ip filter rules")
		return nil
	}

	for i := range rules {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), formatAccessRule(rules[i]))
	}

	return nil

}

func UpsertIPFilter(cmd *cobra.Command, cfg *configs.Configs, status int16, field, value string) error {

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return err
	}

	target, format, msgArgs, err := buildRuleTarget(service, field, value)
	if err != nil {
		return err
	}

	if err := service.UpsertRule(target, status); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", append([]any{ruleAction(status)}, msgArgs...)...)

	return nil

}

func DeleteIPFilter(cmd *cobra.Command, cfg *configs.Configs, status int16, field, value string) error {

	service, err := newAccessRuleService(cfg)
	if err != nil {
		return err
	}

	target, format, msgArgs, err := buildRuleTarget(service, field, value)
	if err != nil {
		return err
	}

	if err := service.DeleteRule(target, status); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted "+format+"\n", append([]any{ruleAction(status)}, msgArgs...)...)

	return nil

}

func buildRuleTarget(service *services.AccessRule, field, value string) (services.RuleTarget, string, []any, error) {

	if isCategoryField(field) {
		target, err := service.NewCategoryRuleTarget(field)
		if err != nil {
			return services.RuleTarget{}, "", nil, err
		}
		return target, "%s category %s", []any{field}, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return services.RuleTarget{}, "", nil, fmt.Errorf("value is required for field %q", field)
	}

	if field == "ip" {
		ip, err := utils.NormalizeIP(value)
		if err != nil {
			return services.RuleTarget{}, "", nil, err
		}
		value = *ip
	}

	target, err := service.NewRuleTarget(field, value)
	if err != nil {
		return services.RuleTarget{}, "", nil, err
	}
	return target, "%s %s %s", []any{field, value}, nil

}

func isCategoryField(field string) bool {
	switch field {
	case "ALL", "LOCAL", "REMOTE":
		return true
	default:
		return false
	}
}

func ruleAction(status int16) string {
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
