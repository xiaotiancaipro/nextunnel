package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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

	cfg := cli.LoadServerConfig(cmd)
	service, err := cli.NewAccessRuleFromConfig(cfg)
	sharedcli.ExitOnErr(cmd, err)

	rules, err := service.ListRules()
	sharedcli.ExitOnErr(cmd, err)

	if len(rules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ip filter rules")
		return
	}

	for i := range rules {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), cli.FormatAccessRule(rules[i]))
	}

}
