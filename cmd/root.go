package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/internal"
)

var geoRules = []args.GeoRule{
	{FlagName: "ip-allow", Status: 1, Field: "ip"},
	{FlagName: "ip-block", Status: 0, Field: "ip"},
	{FlagName: "country-allow", Status: 1, Field: "country"},
	{FlagName: "country-block", Status: 0, Field: "country"},
	{FlagName: "region-allow", Status: 1, Field: "region"},
	{FlagName: "region-block", Status: 0, Field: "region"},
	{FlagName: "city-allow", Status: 1, Field: "city"},
	{FlagName: "city-block", Status: 0, Field: "city"},
}

var categoryRules = []args.CategoryRule{
	{FlagName: "block-all", Status: 0, Category: "ALL"},
	{FlagName: "allow-all", Status: 1, Category: "ALL"},
	{FlagName: "block-local", Status: 0, Category: "LOCAL"},
	{FlagName: "allow-local", Status: 1, Category: "LOCAL"},
}

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

	for i := range geoRules {
		ran, err = geoRules[i].New(cmd, configs)
		if err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
		if ran {
			return
		}
	}

	for i := range categoryRules {
		ran, err = categoryRules[i].New(cmd, configs)
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
