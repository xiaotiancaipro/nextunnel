package ip_filter

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

func NewDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]",
		Short: "delete IP filtering rules",
		Args:  cobra.MaximumNArgs(1),
		Run:   deleteRun,
	}
	setFlags(c)
	return c
}

func deleteRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	status, field, value, err := parseIPFilterFlags(cmd, args)
	utils.ExitOnErr(cmd, err)
	utils.ExitOnErr(cmd, utils.DeleteIPFilter(cmd, cfg, status, field, value))
}
