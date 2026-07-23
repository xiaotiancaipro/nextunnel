package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/ip_filter"
)

func NewAddCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "add IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		RunE:  addRun,
	}
	ip_filter.SetFlags(c)
	return c
}

func addRun(cmd *cobra.Command, args []string) error {

	cfg, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}

	status, field, value, err := ip_filter.ParseIPFilterFlags(cmd, args)
	if err != nil {
		return err
	}

	service, err := cli.NewAccessRuleFromConfig(cfg)
	if err != nil {
		return err
	}
	defer cli.CloseDatabase(service.Database)

	target, format, msgArgs, err := ip_filter.BuildRuleTarget(service, field, value)
	if err != nil {
		return err
	}

	if err := service.UpsertRule(target, status); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", append([]any{ip_filter.RuleAction(status)}, msgArgs...)...)

	return nil

}
