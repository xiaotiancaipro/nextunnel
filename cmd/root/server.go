package root

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services/server"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

const serverDaemonPidEnvKey = "NEXTUNNEL_SERVER_PIDFILE"

type Server struct {
	Configs    *configs.ServerConfigs
	ConfigPath string
}

func NewServer() *cobra.Command {
	fc := func(cmd *cobra.Command, args []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		daemonOp, err2 := cmd.Flags().GetString("daemon")
		pidFlag, err3 := cmd.Flags().GetString("pid-file")
		if err := errors.Join(err1, err2, err3); err != nil {
			cmd.PrintErrf("Invalid flags: %v\n", err)
			os.Exit(1)
		}

		daemonOp = strings.ToLower(strings.TrimSpace(daemonOp))
		switch daemonOp {
		case "":
		case "start", "stop", "reload":
		default:
			cmd.PrintErrf("--daemon must be start, stop, or reload\n")
			os.Exit(1)
		}

		pidPath := utils.ResolvePidPath(configFile, pidFlag)

		switch daemonOp {
		case "start":
			if err := runServerDaemonStart(configFile, pidPath); err != nil {
				cmd.PrintErrf("Daemon start failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "nextunnel server started (pid file %s, log %s)\n", pidPath, utils.LogPathBesideConfig(configFile))
			return
		case "stop":
			if err := runServerDaemonStop(pidPath); err != nil {
				cmd.PrintErrf("Daemon stop failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "nextunnel server stop signal sent (SIGTERM)")
			return
		case "reload":
			if err := runServerDaemonReload(pidPath); err != nil {
				cmd.PrintErrf("Daemon reload failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "nextunnel server reload signal sent (SIGHUP)")
			return
		}

		configs_, err := configs.NewServer(configFile)
		if err != nil {
			cmd.PrintErrf("Failed to load server config, %v\n", err)
			os.Exit(1)
		}
		srv := &Server{Configs: configs_, ConfigPath: configFile}
		if err := srv.Run(); err != nil {
			cmd.PrintErrf("Server error, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "server",
		Short: "Start or manage nextunnel server (foreground, or --daemon start|stop|reload)",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "server.toml", "Path to server config file")
	c.Flags().String("daemon", "", "Daemon control on Unix: start (background), stop (SIGTERM), reload (SIGHUP)")
	c.Flags().String("pid-file", "", "PID file path (default: <config>.pid next to config)")
	return c
}

func (s *Server) Run() error {

	logger := logger_.NewLogger("server")
	if envPid := os.Getenv(serverDaemonPidEnvKey); envPid != "" {
		defer removeServerPidFileIfSelf(envPid)
	}
	srv, err := s.startServer(s.Configs, logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reloadServer := func(source string) {
		cfg, err := configs.NewServer(s.ConfigPath)
		if err != nil {
			logger.Warnf("%s: failed to reload config: %v", source, err)
			return
		}
		logger.Infof("%s: applying zero-downtime server config reload", source)
		if err := srv.ApplyConfig(cfg); err != nil {
			logger.Errorf("%s: failed to reload server config: %v", source, err)
			return
		}
		s.Configs = cfg
		logger.Infof("%s: server config reloaded successfully", source)
	}

	go utils.WatchConfigChanges(ctx, s.ConfigPath, reloadServer, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		if sig == syscall.SIGHUP {
			reloadServer("SIGHUP")
			continue
		}
		logger.Infof("Received signal %v, server is shutting down", sig)
		cancel()
		srv.Stop()
		logger.Infof("Server has stopped")
		return nil
	}

}

func (s *Server) startServer(cfg *configs.ServerConfigs, logger *logrus.Logger) (*server.Server, error) {

	if !cfg.TLS.Enabled {
		logger.Warn("TLS is disabled; control and work connections will be transmitted in plaintext. Do not expose this server directly to untrusted networks.")
	}

	srv, err := server.NewServer(&server.Params{
		BindPort: cfg.BindPort,
		Token:    cfg.Token,
		TLS: configs.ServerTLSConfigs{
			Enabled:  cfg.TLS.Enabled,
			CAFile:   cfg.TLS.CAFile,
			CertFile: cfg.TLS.CertFile,
			KeyFile:  cfg.TLS.KeyFile,
		},
		IPFilter: configs.ServerIPFilterConfigs{
			Allow: cfg.IPFilter.Allow,
			Deny:  cfg.IPFilter.Deny,
		},
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}
	if err := srv.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	logger.Infof("Server started successfully, listening on port: %d, tls=%t", cfg.BindPort, cfg.TLS.Enabled)
	return srv, nil

}
