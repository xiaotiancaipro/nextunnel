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
	c.Flags().String("ip-filter-allow-ip", "", "ip allow")
	c.Flags().String("ip-filter-block-ip", "", "ip block")
	c.Flags().String("ip-filter-allow-country", "", "country allow")
	c.Flags().String("ip-filter-block-country", "", "country block")
	c.Flags().String("ip-filter-allow-region", "", "region allow")
	c.Flags().String("ip-filter-block-region", "", "region block")
	c.Flags().String("ip-filter-allow-city", "", "city allow")
	c.Flags().String("ip-filter-block-city", "", "city block")
	c.Flags().Bool("ip-filter-block-all", false, "block all connections")
	c.Flags().Bool("ip-filter-allow-all", false, "allow all connections")
	c.Flags().Bool("ip-filter-block-local", false, "block local network connections")
	c.Flags().Bool("ip-filter-allow-local", false, "allow local network connections")
	c.Flags().Bool("ip-filter-block-remote", false, "block remote (non-local) network connections")
	c.Flags().Bool("ip-filter-allow-remote", false, "allow remote (non-local) network connections")
	return c
}

func (c *root) run(cmd *cobra.Command, _ []string) {

	cfg := args.LoadConfig(cmd)

	ran, err := args.RunGenerateCerts(cmd, cfg)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	if ran {
		return
	}

	ran, err = args.RunIPFilters(cmd, cfg)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	if ran {
		return
	}

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
