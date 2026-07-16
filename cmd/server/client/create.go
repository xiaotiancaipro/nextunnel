package client

import (
	"fmt"

	"github.com/spf13/cobra"
	utils "github.com/xiaotiancaipro/nextunnel/internal/server/utils/cli"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
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
	cfg := shared.LoadServerConfig(cmd)
	registry, err := utils.NewClientRegistryFromConfig(cfg)
	shared.ExitOnErr(cmd, err)
	client, err := registry.Create(args[0], portStart, portEnd)
	shared.ExitOnErr(cmd, err)
	if portStart > 0 && portEnd > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=%d-%d)\n", client.Name, client.Id, portStart, portEnd)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=all)\n", client.Name, client.Id)
	}
}
