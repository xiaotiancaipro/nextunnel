package root

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services/client"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
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
		srv := &Client{Configs: configs_}
		if err := srv.Run(); err != nil {
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

	logger := logger_.NewLogger("client")
	if !c.Configs.TLS.Enabled {
		logger.Warn("TLS is disabled; credentials and tunneled traffic may be exposed on the network. Only use this mode in trusted environments.")
	}

	proxies := make([]configs.ProxyConfig, 0, len(c.Configs.Proxies))
	for _, p := range c.Configs.Proxies {
		proxies = append(proxies, configs.ProxyConfig{
			Name:       p.Name,
			Type:       p.Type,
			RemotePort: p.RemotePort,
			LocalIP:    utils.LocalIP(p.LocalIP),
			LocalPort:  p.LocalPort,
		})
	}

	srv, err := client.NewClient(&client.Params{
		ServerAddr: c.Configs.ServerAddr,
		ServerPort: c.Configs.ServerPort,
		Token:      c.Configs.Token,
		TLS: configs.ClientTLSConfigs{
			Enabled:            c.Configs.TLS.Enabled,
			ServerName:         c.Configs.TLS.ServerName,
			CAFile:             c.Configs.TLS.CAFile,
			CertFile:           c.Configs.TLS.CertFile,
			KeyFile:            c.Configs.TLS.KeyFile,
			InsecureSkipVerify: c.Configs.TLS.InsecureSkipVerify,
		},
		Proxies: proxies,
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}
	logger.Infof("Client started successfully, connected to server: %s:%d, tls=%t", c.Configs.ServerAddr, c.Configs.ServerPort, c.Configs.TLS.Enabled)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("Received signal %v, client is shutting down", sig)

	srv.Stop()
	logger.Infof("Client has stopped")

	return nil

}
