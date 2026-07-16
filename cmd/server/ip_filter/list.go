package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	utils "github.com/xiaotiancaipro/nextunnel/internal/server/utils/cli"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func NewListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list current IP filtering rules",
		Args:  cobra.NoArgs,
		Run:   listRun,
	}
	return c
}

func listRun(cmd *cobra.Command, _ []string) {
	cfg := shared.LoadServerConfig(cmd)
	service, err := utils.NewAccessRuleFromConfig(cfg)
	shared.ExitOnErr(cmd, err)
	rules, err := service.ListRules()
	shared.ExitOnErr(cmd, err)
	if len(rules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ip filter rules")
		return
	}
	for i := range rules {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), utils.FormatAccessRule(rules[i]))
	}
}
