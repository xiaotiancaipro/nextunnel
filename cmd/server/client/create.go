package client

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
)

var (
	portStart int
	portEnd   int
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client access record",
		Args:  cobra.ExactArgs(1),
		RunE:  createRun,
	}
	cmd.Flags().IntVar(&portStart, "port-start", 0, "inclusive start of allocated remote port range")
	cmd.Flags().IntVar(&portEnd, "port-end", 0, "inclusive end of allocated remote port range")
	return cmd
}

func createRun(cmd *cobra.Command, args []string) error {

	cfg, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}

	registry, err := cli.NewClientRegistryFromConfig(cfg)
	if err != nil {
		return err
	}
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.Create(args[0], portStart, portEnd)
	if err != nil {
		return err
	}

	if portStart > 0 && portEnd > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=%d-%d)\n", client.Name, client.Id, portStart, portEnd)
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=all)\n", client.Name, client.Id)

	return nil

}
