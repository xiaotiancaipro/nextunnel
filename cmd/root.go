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
	c.Flags().String("config", "nextunnel-server.toml", "configuration file Path")
	c.Flags().String("generate-certs", "", "client certificate generation path")
	c.Flags().String("ip-allow", "", "ip allow")
	c.Flags().String("ip-block", "", "ip block")
	c.Flags().String("country-allow", "", "country allow")
	c.Flags().String("country-block", "", "country block")
	c.Flags().String("region-allow", "", "region allow")
	c.Flags().String("region-block", "", "region block")
	c.Flags().String("city-allow", "", "city allow")
	c.Flags().String("city-block", "", "city block")
	c.Flags().Bool("block-all", false, "block all connections")
	c.Flags().Bool("allow-all", false, "allow all connections")
	c.Flags().Bool("block-local", false, "block local network connections")
	c.Flags().Bool("allow-local", false, "allow local network connections")
	c.Flags().Bool("block-remote", false, "block remote (non-local) network connections")
	c.Flags().Bool("allow-remote", false, "allow remote (non-local) network connections")
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

	for i := range args.IPFilterRules {
		ran, err = args.IPFilterRules[i].New(cmd, configs)
		if err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
		if ran {
			return
		}
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
