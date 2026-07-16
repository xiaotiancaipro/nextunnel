package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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
		shared.ExitOnErr(cmd, fmt.Errorf("client name is required"))
	}

	cfg := cli.LoadServerConfig(cmd)
	registry, certService, err := cli.NewClientRegistryAndCertFromConfig(cfg)
	shared.ExitOnErr(cmd, err)

	client, err := registry.GetByName(clientName)
	shared.ExitOnErr(cmd, err)

	certID, err := services.ParseCertID(args[1])
	shared.ExitOnErr(cmd, err)

	if err := certService.Delete(client.Id, certID); err != nil {
		shared.ExitOnErr(cmd, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)

}
