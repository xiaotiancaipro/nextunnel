package client

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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
		Run:   createRun,
	}
	cmd.Flags().IntVar(&portStart, "port-start", 0, "inclusive start of allocated remote port range")
	cmd.Flags().IntVar(&portEnd, "port-end", 0, "inclusive end of allocated remote port range")
	return cmd
}

func createRun(cmd *cobra.Command, args []string) {

	cfg := cli.LoadServerConfig(cmd)
	registry, err := cli.NewClientRegistryFromConfig(cfg)
	sharedcli.ExitOnErr(cmd, err)
	defer cli.CloseDatabase(registry.Database)

	client, err := registry.Create(args[0], portStart, portEnd)
	cli.ExitOnDBErr(cmd, err, registry.Database)

	if portStart > 0 && portEnd > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=%d-%d)\n", client.Name, client.Id, portStart, portEnd)
		return
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=all)\n", client.Name, client.Id)

}
