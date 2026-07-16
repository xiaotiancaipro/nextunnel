package cert

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

func NewDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [name] [cert-id]",
		Short: "delete a client TLS certificate",
		Args:  cobra.ExactArgs(2),
		Run:   deleteRun,
	}
	return c
}

func deleteRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.DeleteClientCert(cmd, cfg, args[0], args[1]))
}
