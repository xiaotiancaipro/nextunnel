package root

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/shared"
	"github.com/xiaotiancaipro/nextunnel/internal/client"
)

type Root struct{}

func (r *Root) New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-client",
		Short:   "nextunnel-client",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     r.run,
	}
	c.Flags().StringP("config", "c", shared.ClientDefaultConfigPath, "Configuration File Path")
	return c
}

func (r *Root) run(cmd *cobra.Command, _ []string) {
	configs := shared.LoadClientConfig(cmd)
	app, err := client.NewApp(configs)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	shared.Run(cmd, app)
}
