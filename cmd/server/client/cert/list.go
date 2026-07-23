package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/client/cert"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

func NewListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list [name]",
		Short: "list certificates for a client",
		Args:  cobra.ExactArgs(1),
		RunE:  listRun,
	}
	return c
}

func listRun(cmd *cobra.Command, args []string) error {

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

	items, err := certService.List(client.Id)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "no certificates for client %q\n", clientName)
		return nil
	}

	for _, item := range items {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tcreated=%s\texpires=%s\tserial=%s\n",
			item.ID,
			sharedtimezone.FormatUTC(item.CreatedAt),
			cert.FormatExpires(item.ExpiresAt),
			item.Serial,
		)
	}

	return nil

}
