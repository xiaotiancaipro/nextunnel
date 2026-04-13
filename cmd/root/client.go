package root

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

type Client struct {
	Configs *configs.ClientConfigs
}

func NewClient() *cobra.Command {
	fc := func(cmd *cobra.Command, _ []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		configs_, err2 := configs.NewClient(configFile)
		if err := errors.Join(err1, err2); err != nil {
			cmd.PrintErrf("Failed to load client config, %v\n", err)
			os.Exit(1)
		}
		client := &Client{Configs: configs_}
		if err := client.Run(); err != nil {
			cmd.PrintErrf("Client error, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "client",
		Short: "Start nextunnel client",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "client.toml", "Path to client config file")
	return c
}

func (c *Client) Run() error {

	logger := utils.NewLogger("client")

	proxies := make([]services.ProxyConfig, 0, len(c.Configs.Proxies))
	for _, p := range c.Configs.Proxies {
		proxies = append(proxies, services.ProxyConfig{
			Name:       p.Name,
			Type:       p.Type,
			RemotePort: p.RemotePort,
			LocalIP:    utils.LocalIP(p.LocalIP),
			LocalPort:  p.LocalPort,
		})
	}

	client, err := services.NewClient(&services.ClientParams{
		ServerAddr: c.Configs.ServerAddr,
		ServerPort: c.Configs.ServerPort,
		Token:      c.Configs.Token,
		TLS: services.ClientTLSConfig{
			Enabled:            c.Configs.TLS.Enabled,
			ServerName:         c.Configs.TLS.ServerName,
			CAFile:             c.Configs.TLS.CAFile,
			InsecureSkipVerify: c.Configs.TLS.InsecureSkipVerify,
		},
		Proxies: proxies,
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	if err := client.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}
	logger.Infof("Client started successfully, connected to server: %s:%d, tls=%t", c.Configs.ServerAddr, c.Configs.ServerPort, c.Configs.TLS.Enabled)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("Received signal %v, client is shutting down", sig)

	client.Stop()
	logger.Infof("Client has stopped")

	return nil

}
