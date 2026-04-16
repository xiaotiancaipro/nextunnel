package root

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	configs_ "github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/client/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/utils/logger"
)

type client struct {
	workdir string
	pidFile string
	configs *configs_.ClientConfigs
	logger  *logrus.Logger
}

func NewClient() *cobra.Command {
	c := &cobra.Command{
		Use:   "client",
		Short: "Manage nextunnel client",
		Args:  cobra.ExactArgs(0),
		Run:   new(client).run,
	}
	c.Flags().StringP("workdir", "w", ".nextunnel-client", "Working directory")
	c.Flags().StringP("daemon", "d", "", "Daemon control: start, stop, reload")
	return c
}

func (c *client) run(cmd *cobra.Command, _ []string) {

	workdir, err1 := cmd.Flags().GetString("workdir")
	daemonOp, err2 := cmd.Flags().GetString("daemon")
	if err := errors.Join(err1, err2); err != nil {
		utils.NotifyDaemonStartFailure(fmt.Errorf("invalid flags: %w", err))
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	configFile := path.Join(workdir, "nextunnel.toml")
	logFile := path.Join(workdir, "logs", "nextunnel.log")

	c.workdir = workdir
	c.pidFile = path.Join(workdir, "nextunnel.pid")

	daemonOp = strings.ToLower(strings.TrimSpace(daemonOp))
	switch daemonOp {
	case "":
	case "start":
		if err := c.daemonStart(); err != nil {
			cmd.PrintErrf("Daemon start failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Printf("nextunnel client started (pid file %s, log %s)\n", c.pidFile, logFile)
		return
	case "stop":
		if err := c.daemonStop(); err != nil {
			cmd.PrintErrf("Daemon stop failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel client stop signal sent (SIGTERM)")
		return
	case "reload":
		if err := c.daemonReload(); err != nil {
			cmd.PrintErrf("Daemon reload failed: %v\n", err)
			os.Exit(1)
		}
		cmd.Println("nextunnel client reload signal sent (SIGHUP)")
		return
	default:
		utils.NotifyDaemonStartFailure(errors.New("--daemon must be start, stop, or reload"))
		cmd.PrintErrf("--daemon must be start, stop, or reload\n")
		os.Exit(1)
	}

	configs, err := configs_.NewClient(configFile)
	if err != nil {
		utils.NotifyDaemonStartFailure(fmt.Errorf("load client config: %w", err))
		cmd.PrintErrf("Failed to load client config, %v\n", err)
		os.Exit(1)
	}

	logger, err := logger_.New(logFile)
	if err != nil {
		utils.NotifyDaemonStartFailure(fmt.Errorf("init logger: %w", err))
		cmd.PrintErrf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	c.configs = configs
	c.logger = logger

	srv, err := c.start()
	if err != nil {
		utils.NotifyDaemonStartFailure(err)
		cmd.PrintErrf("Failed to start client: %v\n", err)
		os.Exit(1)
	}
	utils.NotifyDaemonReady()

	var mu sync.Mutex
	var drainingClients []*services.Client
	stopped := false

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {

		sig := <-sigCh

		if sig == syscall.SIGHUP {
			configsNew, err := configs_.NewClient(configFile)
			if err != nil {
				logger.Warnf("Failed to reload config: %v", err)
				continue
			}
			mu.Lock()
			if stopped {
				mu.Unlock()
				continue
			}
			logger.Info("Applying zero-downtime client config reload")
			c.configs = configsNew
			nextClient, err := c.start()
			if err != nil {
				logger.Errorf("Failed to start client with new config: %v", err)
				mu.Unlock()
				continue
			}
			prevClient := srv
			srv = nextClient
			configs = configsNew
			if prevClient != nil {
				prevClient.Retire()
				drainingClients = append(drainingClients, prevClient)
				time.AfterFunc(5*time.Second, prevClient.Stop)
			}
			mu.Unlock()
			logger.Info("Client config reloaded successfully")
			continue
		}

		logger.Infof("Received signal %v, client is shutting down", sig)
		mu.Lock()
		stopped = true
		if srv != nil {
			srv.Stop()
		}
		for _, draining := range drainingClients {
			draining.Stop()
		}
		mu.Unlock()
		logger.Infof("Client has stopped")

		os.Exit(0)

	}

}

func (c *client) start() (*services.Client, error) {

	if !c.configs.TLS.Enabled {
		c.logger.Warn("TLS is disabled; credentials and tunneled traffic may be exposed on the network. Only use this mode in trusted environments.")
	}

	proxies := make([]configs_.ProxyConfig, 0, len(c.configs.Proxies))
	for _, p := range c.configs.Proxies {
		proxies = append(proxies, configs_.ProxyConfig{
			Name:       p.Name,
			Type:       p.Type,
			RemotePort: p.RemotePort,
			LocalIP:    utils.LocalIP(p.LocalIP),
			LocalPort:  p.LocalPort,
		})
	}

	params := &services.Params{
		ClientID:   c.configs.ClientID,
		ServerAddr: c.configs.ServerAddr,
		ServerPort: c.configs.ServerPort,
		Token:      c.configs.Token,
		TLS:        c.configs.TLS,
		Proxies:    proxies,
		Logger:     c.logger,
	}
	srv, err := services.NewClient(params)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	if err := srv.Start(); err != nil {
		return nil, fmt.Errorf("failed to start client: %w", err)
	}

	c.logger.Infof("Client started successfully, client_id=%s, connected to server: %s:%d, tls=%t", c.configs.ClientID, c.configs.ServerAddr, c.configs.ServerPort, c.configs.TLS.Enabled)
	return srv, nil

}

func (c *client) daemonStart() error {

	if err := utils.EnsureStalePidFileCleared(c.pidFile); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	absExe, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	readyR, readyW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create daemon readiness pipe: %w", err)
	}
	defer func() { _ = readyR.Close() }()

	cmd := exec.Command(absExe, "client", "--workdir", c.workdir)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.ExtraFiles = []*os.File{readyW}
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", utils.DaemonReadyEnv, 3))

	if err := cmd.Start(); err != nil {
		_ = readyW.Close()
		return fmt.Errorf("start daemon process: %w", err)
	}
	_ = readyW.Close()

	pid := cmd.Process.Pid
	if err := utils.WritePidFile(c.pidFile, pid); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return fmt.Errorf("write pid file: %w", err)
	}

	if err := utils.AwaitDaemonReady(readyR); err != nil {
		_ = os.Remove(c.pidFile)
		if utils.ProcessAlive(pid) {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
		return fmt.Errorf("daemon failed to become ready: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release process: %w", err)
	}

	return nil

}

func (c *client) daemonStop() error {
	pid, err := utils.ReadPidFile(c.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !utils.ProcessAlive(pid) {
		_ = os.Remove(c.pidFile)
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal TERM to pid %d: %w", pid, err)
	}
	return nil
}

func (c *client) daemonReload() error {
	pid, err := utils.ReadPidFile(c.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pid file not found: %s", c.pidFile)
		}
		return fmt.Errorf("read pid file: %w", err)
	}
	if !utils.ProcessAlive(pid) {
		return fmt.Errorf("no process with pid %d", pid)
	}
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("signal HUP to pid %d: %w", pid, err)
	}
	return nil
}
