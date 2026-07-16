package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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

	client, err := registry.GetByName(clientName)
	sharedcli.ExitOnErr(cmd, err)

	certID, err := services.ParseCertID(args[1])
	sharedcli.ExitOnErr(cmd, err)

	if err := certService.Delete(client.Id, certID); err != nil {
		sharedcli.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)

}
