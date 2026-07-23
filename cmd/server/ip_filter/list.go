package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/ip_filter"
)

func NewListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list current IP filtering rules",
		Args:  cobra.NoArgs,
		RunE:  listRun,
	}
	return c
}

func listRun(cmd *cobra.Command, _ []string) error {

	cfg, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}

	service, err := cli.NewAccessRuleFromConfig(cfg)
	if err != nil {
		return err
	}
	defer cli.CloseDatabase(service.Database)

	rules, err := service.ListRules()
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ip filter rules")
		return nil
	}

	for i := range rules {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), ip_filter.FormatAccessRule(rules[i]))
	}

	return nil

}
