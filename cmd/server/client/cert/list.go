package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
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

	clientName := strings.TrimSpace(args[0])
	if clientName == "" {
		sharedcli.ExitOnErr(cmd, fmt.Errorf("client name is required"))
	}

	cfg := cli.LoadServerConfig(cmd)
	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	sharedcli.ExitOnErr(cmd, err)
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.GetByName(clientName)
	cli.ExitOnDBErr(cmd, err, registry.Database)

	items, err := certService.List(client.Id)
	cli.ExitOnDBErr(cmd, err, registry.Database)

	if len(items) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "no certificates for client %q\n", clientName)
		return
	}

	for _, item := range items {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tcreated=%s\texpires=%s\tserial=%s\n",
			item.ID,
			sharedtimezone.FormatUTC(item.CreatedAt),
			cli.FormatExpires(item.ExpiresAt),
			item.Serial,
		)
	}

}
