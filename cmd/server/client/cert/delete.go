package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
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

	certID, err := sharedstring.ParseUUID(args[1])
	cli.ExitOnDBErr(cmd, err, registry.Database)

	if err := certService.Delete(client.Id, certID); err != nil {
		cli.ExitOnDBErr(cmd, err, registry.Database)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)

}
