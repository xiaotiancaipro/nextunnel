package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/utils"
	"github.com/xiaotiancaipro/nextunnel-server/internal"
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
	c.PersistentFlags().String("config", "nextunnel-server.toml", "configuration file path")
	c.AddCommand(new(client).new())
	c.AddCommand(new(ipFilter).new())
	return c
}

func (r *Root) run(cmd *cobra.Command, _ []string) {

	cfg := utils.LoadConfig(cmd)

	app, err := internal.NewApp(cfg)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err = <-errCh:
		signal.Stop(sigCh)
		if err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
		return
	case <-sigCh:
		signal.Stop(sigCh)
		app.Stop()
		if err = <-errCh; err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
		return
	}

}
