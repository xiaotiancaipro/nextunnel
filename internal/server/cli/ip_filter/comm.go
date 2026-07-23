package ip_filter

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
)

const (
	actionAllow = "allowed"
	actionBlock = "blocked"
)

var ipFilterFields = []ipFilterField{
	{flag: "ip", field: "ip", needsValue: true},
	{flag: "country", field: "country", needsValue: true},
	{flag: "region", field: "region", needsValue: true},
	{flag: "city", field: "city", needsValue: true},
	{flag: "all", field: "ALL", needsValue: false},
	{flag: "local", field: "LOCAL", needsValue: false},
	{flag: "remote", field: "REMOTE", needsValue: false},
}

type ipFilterField struct {
	flag       string
	field      string
	needsValue bool
}

func SetFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("allow", false, "allow matching traffic")
	cmd.Flags().Bool("block", false, "block matching traffic")
	cmd.Flags().Bool("ip", false, "match by IP address (requires value)")
	cmd.Flags().Bool("country", false, "match by country (requires value)")
	cmd.Flags().Bool("region", false, "match by region (requires value)")
	cmd.Flags().Bool("city", false, "match by city (requires value)")
	cmd.Flags().Bool("all", false, "match all traffic")
	cmd.Flags().Bool("local", false, "match local network traffic")
	cmd.Flags().Bool("remote", false, "match remote network traffic")
}

func ParseIPFilterFlags(cmd *cobra.Command, posArgs []string) (status int16, field, value string, err error) {

	allow, _ := cmd.Flags().GetBool("allow")
	block, _ := cmd.Flags().GetBool("block")
	switch {
	case allow && block:
		return 0, "", "", fmt.Errorf("specify exactly one of --allow or --block")
	case !allow && !block:
		return 0, "", "", fmt.Errorf("specify one of --allow or --block")
	case allow:
		status = 1
	default:
		status = 0
	}

	var selected *ipFilterField
	for i := range ipFilterFields {
		set, _ := cmd.Flags().GetBool(ipFilterFields[i].flag)
		if !set {
			continue
		}
		if selected != nil {
			return 0, "", "", fmt.Errorf("specify exactly one of --ip, --country, --region, --city, --all, --local, --remote")
		}
		selected = new(ipFilterFields[i])
	}
	if selected == nil {
		return 0, "", "", fmt.Errorf("specify one of --ip, --country, --region, --city, --all, --local, --remote")
	}

	field = selected.field
	if selected.needsValue {
		if len(posArgs) != 1 {
			return 0, "", "", fmt.Errorf("value is required for --%s", selected.flag)
		}
		value = posArgs[0]
	} else if len(posArgs) > 0 {
		return 0, "", "", fmt.Errorf("value must not be set for --%s", selected.flag)
	}

	return status, field, value, nil

}

func BuildRuleTarget(service *services.AccessRule, field, value string) (services.RuleTarget, string, []any, error) {
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
		ip, err := sharednetwork.NormalizeIP(value)
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

func RuleAction(status int16) string {
	if status == 1 {
		return actionAllow
	}
	return actionBlock
}

func FormatAccessRule(rule models.AccessRule) string {
	action := actionBlock
	if rule.Status == 1 {
		action = actionAllow
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

func isCategoryField(field string) bool {
	switch field {
	case "ALL", "LOCAL", "REMOTE":
		return true
	default:
		return false
	}
}
