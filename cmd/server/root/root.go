package root

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/shared"
	"github.com/xiaotiancaipro/nextunnel/internal/server"
)

type Root struct{}

func (r *Root) New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-server",
		Short:   "nextunnel-server",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     r.run,
	}
	c.PersistentFlags().String("config", shared.ServerDefaultConfigPath, "configuration file path (overrides $"+shared.ServerEnvConfigPath+")")
	c.AddCommand(new(client).new())
	c.AddCommand(new(ipFilter).new())
	return c
}

func (r *Root) run(cmd *cobra.Command, _ []string) {
	cfg := shared.LoadServerConfig(cmd)
	app, err := server.NewApp(cfg, cmd.Version)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	shared.Run(cmd, app)
}
