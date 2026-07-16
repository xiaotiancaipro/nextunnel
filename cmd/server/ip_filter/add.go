package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func NewAddCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "add IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		Run:   addRun,
	}
	cli.SetFlags(c)
	return c
}

func addRun(cmd *cobra.Command, args []string) {

	cfg := cli.LoadServerConfig(cmd)
	status, field, value, err := cli.ParseIPFilterFlags(cmd, args)
	shared.ExitOnErr(cmd, err)

	service, err := cli.NewAccessRuleFromConfig(cfg)
	shared.ExitOnErr(cmd, err)

	target, format, msgArgs, err := cli.BuildRuleTarget(service, field, value)
	shared.ExitOnErr(cmd, err)

	if err := service.UpsertRule(target, status); err != nil {
		shared.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", append([]any{cli.RuleAction(status)}, msgArgs...)...)

}
