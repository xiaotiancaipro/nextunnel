package ip_filter

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
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
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.ListIPFilters(cmd, cfg))
}
