package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/client/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

const clientReloadDrainGrace = 5 * time.Second

type Client struct {
	Configs    *configs.ClientConfigs
	ConfigPath string
}

func NewClient() *cobra.Command {
	fc := func(cmd *cobra.Command, _ []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		configs_, err2 := configs.NewClient(configFile)
		if err := errors.Join(err1, err2); err != nil {
			cmd.PrintErrf("Failed to load client config, %v\n", err)
			os.Exit(1)
		}
		srv := &Client{Configs: configs_, ConfigPath: configFile}
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

	logger := logger_.New("client")
	srv, err := c.startClient(c.Configs, logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	currentClient := srv
	var drainingClients []*services.Client
	stopped := false

	reloadClient := func(source string) {
		cfg, err := configs.NewClient(c.ConfigPath)
		if err != nil {
			logger.Warnf("%s: failed to reload config: %v", source, err)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return
		}

		prevClient := currentClient
		logger.Infof("%s: applying zero-downtime client config reload", source)

		nextClient, err := c.startClient(cfg, logger)
		if err != nil {
			logger.Errorf("%s: failed to start client with new config: %v", source, err)
			return
		}

		currentClient = nextClient
		c.Configs = cfg
		if prevClient != nil {
			prevClient.Retire()
			drainingClients = append(drainingClients, prevClient)
			time.AfterFunc(clientReloadDrainGrace, prevClient.Stop)
		}
		logger.Infof("%s: client config reloaded successfully", source)
	}

	go utils.WatchConfigChanges(ctx, c.ConfigPath, reloadClient, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		if sig == syscall.SIGHUP {
			reloadClient("SIGHUP")
			continue
		}
		logger.Infof("Received signal %v, client is shutting down", sig)
		mu.Lock()
		stopped = true
		cancel()
		if currentClient != nil {
			currentClient.Stop()
		}
		for _, draining := range drainingClients {
			draining.Stop()
		}
		mu.Unlock()
		logger.Infof("Client has stopped")
		return nil
	}

}

func (c *Client) startClient(cfg *configs.ClientConfigs, logger *logrus.Logger) (*services.Client, error) {

	if !cfg.TLS.Enabled {
		logger.Warn("TLS is disabled; credentials and tunneled traffic may be exposed on the network. Only use this mode in trusted environments.")
	}

	proxies := make([]configs.ProxyConfig, 0, len(cfg.Proxies))
	for _, p := range cfg.Proxies {
		proxies = append(proxies, configs.ProxyConfig{
			Name:       p.Name,
			Type:       p.Type,
			RemotePort: p.RemotePort,
			LocalIP:    utils.LocalIP(p.LocalIP),
			LocalPort:  p.LocalPort,
		})
	}

	srv, err := services.NewClient(&services.Params{
		ClientID:   cfg.ClientID,
		ServerAddr: cfg.ServerAddr,
		ServerPort: cfg.ServerPort,
		Token:      cfg.Token,
		TLS: configs.ClientTLSConfigs{
			Enabled:            cfg.TLS.Enabled,
			ServerName:         cfg.TLS.ServerName,
			CAFile:             cfg.TLS.CAFile,
			CertFile:           cfg.TLS.CertFile,
			KeyFile:            cfg.TLS.KeyFile,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		},
		Proxies: proxies,
		Logger:  logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	if err := srv.Start(); err != nil {
		return nil, fmt.Errorf("failed to start client: %w", err)
	}

	logger.Infof("Client started successfully, client_id=%s, connected to server: %s:%d, tls=%t", cfg.ClientID, cfg.ServerAddr, cfg.ServerPort, cfg.TLS.Enabled)
	return srv, nil

}
