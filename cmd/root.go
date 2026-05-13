package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-client/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-client/internal"
)

type root struct{}

func New() *cobra.Command {
	c := &cobra.Command{
		Short:   "nextunnel-client",
		Version: "v0.1.0",
		Args:    cobra.ExactArgs(0),
		Run:     new(root).run,
	}
	c.Flags().StringP("config", "c", "nextunnel-client.toml", "Configuration File Path")
	return c
}

func (c *root) run(cmd *cobra.Command, _ []string) {

	configs := new(args.Config).New(cmd)
	app, err := internal.NewApp(configs)
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
