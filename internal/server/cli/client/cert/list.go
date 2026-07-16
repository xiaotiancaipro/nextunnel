package cert

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

func NewListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list [name]",
		Short: "list certificates for a client",
		Args:  cobra.ExactArgs(1),
		Run:   listRun,
	}
	return c
}

func listRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.ListClientCerts(cmd, cfg, args[0]))
}
