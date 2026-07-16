package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	utils "github.com/xiaotiancaipro/nextunnel/internal/server/utils/cli"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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
	cfg := shared.LoadServerConfig(cmd)
	clientName := strings.TrimSpace(args[0])
	if clientName == "" {
		shared.ExitOnErr(cmd, fmt.Errorf("client name is required"))
	}

	registry, certService, err := utils.NewClientRegistryAndCertFromConfig(cfg)
	shared.ExitOnErr(cmd, err)
	client, err := registry.GetByName(clientName)
	shared.ExitOnErr(cmd, err)
	expiresAt, err := utils.ParseExpiresAt(expiresAt)
	shared.ExitOnErr(cmd, err)
	info, err := certService.Create(client, expiresAt)
	shared.ExitOnErr(cmd, err)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created certificate %q for client %q (expires=%s, serial=%s)\n",
		info.ID, clientName, utils.FormatExpires(info.ExpiresAt), info.Serial)
}
