package client

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/client/args"
	"github.com/xiaotiancaipro/nextunnel/internal/client"
)

type Root struct{}

func (r *Root) New() *cobra.Command {
	c := &cobra.Command{
		Short: "nextunnel-client",
		Args:  cobra.ExactArgs(0),
		Run:   r.run,
	}
	c.Flags().StringP("config", "c", "nextunnel-client.toml", "Configuration File Path")
	return c
}

func (r *Root) run(cmd *cobra.Command, _ []string) {

	configs := new(args.Config).New(cmd)
	app, err := client.NewApp(configs)
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
