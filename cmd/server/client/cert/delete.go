package cert

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedstring "github.com/xiaotiancaipro/nextunnel/internal/shared/string"
)

func NewDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [name] [cert-id]",
		Short: "delete a client TLS certificate",
		Args:  cobra.ExactArgs(2),
		RunE:  deleteRun,
	}
	return c
}

func deleteRun(cmd *cobra.Command, args []string) error {

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

	certID, err := sharedstring.ParseUUID(args[1])
	if err != nil {
		return err
	}

	if err := certService.Delete(client.Id, certID); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)

	return nil

}
