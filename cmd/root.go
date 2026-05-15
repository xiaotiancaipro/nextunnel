package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/internal"
)

type root struct{}

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Short:   "nextunnel-server",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     new(root).run,
	}
	c.Flags().StringP("config", "c", "nextunnel-server.toml", "configuration file Path")
	c.Flags().StringP("generate-certs", "g", "", "output directory for client.crt and client.key (uses [tls] CA from config); ignored if flag not set or path is empty")
	return c
}

func (c *root) run(cmd *cobra.Command, _ []string) {

	configs := new(args.Config).New(cmd)

	ran, err := new(args.GenerateCerts).New(cmd, configs)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	if ran {
		return
	}

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
