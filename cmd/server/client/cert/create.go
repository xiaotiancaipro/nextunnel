package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/client/cert"
)

var expiresAt string

func NewCreateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client TLS certificate",
		Args:  cobra.ExactArgs(1),
		RunE:  createRun,
	}
	c.Flags().StringVar(&expiresAt, "expires-at", "", "certificate expiry time in RFC3339 format (default: never expires)")
	return c
}

func createRun(cmd *cobra.Command, args []string) error {

	clientName := strings.TrimSpace(args[0])
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}

	cfg, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}

	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	if err != nil {
		return err
	}
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.GetByName(clientName)
	if err != nil {
		return err
	}

	expiresAt, err := cert.ParseExpiresAt(expiresAt)
	if err != nil {
		return err
	}

	info, err := certService.Create(client, expiresAt)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"created certificate %q for client %q (expires=%s, serial=%s)\n",
		info.ID,
		clientName,
		cert.FormatExpires(info.ExpiresAt),
		info.Serial,
	)

	return nil

}
