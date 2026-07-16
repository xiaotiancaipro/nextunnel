package cert

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

var expiresAt string

func NewCreateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client TLS certificate",
		Args:  cobra.ExactArgs(1),
		Run:   createRun,
	}
	c.Flags().StringVar(&expiresAt, "expires-at", "", "certificate expiry time in RFC3339 format (default: never expires)")
	return c
}

func createRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.CreateClientCert(cmd, cfg, args[0], expiresAt))
}
