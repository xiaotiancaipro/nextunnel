package ip_filter

import (
	"fmt"

	"github.com/spf13/cobra"
	utils "github.com/xiaotiancaipro/nextunnel/internal/server/utils/cli"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func NewDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "delete IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		Run:   deleteRun,
	}
	utils.SetFlags(c)
	return c
}

func deleteRun(cmd *cobra.Command, args []string) {

	status, field, value, err := utils.ParseIPFilterFlags(cmd, args)
	shared.ExitOnErr(cmd, err)

	cfg := shared.LoadServerConfig(cmd)
	service, err := utils.NewAccessRuleFromConfig(cfg)
	shared.ExitOnErr(cmd, err)

	target, format, msgArgs, err := utils.BuildRuleTarget(service, field, value)
	shared.ExitOnErr(cmd, err)

	if err := service.DeleteRule(target, status); err != nil {
		shared.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted "+format+"\n", append([]any{utils.RuleAction(status)}, msgArgs...)...)

}
