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
			cmd.PrintErrf("加载客户端配置失败, %v\n", err)
			os.Exit(1)
		}
		client := &Client{Configs: configs_}
		if err := client.StartAndStop(); err != nil {
			cmd.PrintErrf("客户端异常, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "client",
		Short: "启动内网穿透客户端 (client)",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "config/nextunnel-client.toml", "客户端配置文件路径")
	return c
}

func (c *Client) StartAndStop() error {

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
		Proxies:    proxies,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %w", err)
	}

	if err := client.Start(); err != nil {
		return fmt.Errorf("启动客户端失败: %w", err)
	}
	logger.Infof("客户端启动成功, 连接服务端: %s:%d", c.Configs.ServerAddr, c.Configs.ServerPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("已收到信号 %v, 客户端正在关闭", sig)

	client.Stop()
	logger.Infof("客户端已关闭")

	return nil

}
