package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func NewDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "delete IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		Run:   deleteRun,
	}
	cli.SetFlags(c)
	return c
}

func deleteRun(cmd *cobra.Command, args []string) {

	status, field, value, err := cli.ParseIPFilterFlags(cmd, args)
	sharedcli.ExitOnErr(cmd, err)

	cfg := cli.LoadServerConfig(cmd)
	service, err := cli.NewAccessRuleFromConfig(cfg)
	sharedcli.ExitOnErr(cmd, err)

	target, format, msgArgs, err := cli.BuildRuleTarget(service, field, value)
	sharedcli.ExitOnErr(cmd, err)

	if err := service.DeleteRule(target, status); err != nil {
		sharedcli.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted "+format+"\n", append([]any{cli.RuleAction(status)}, msgArgs...)...)

}
