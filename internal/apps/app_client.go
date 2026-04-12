package apps

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	client2 "github.com/xiaotiancaipro/nextunnel/internal/services"
)

func (c *Client) StartAndStop() error {

	if c.logger == nil {
		c.logger = newLogger("client")
	}

	proxies := make([]client2.ProxyConfig, 0, len(c.Configs.Proxies))
	for _, p := range c.Configs.Proxies {
		proxies = append(proxies, client2.ProxyConfig{
			Name:       p.Name,
			Type:       p.Type,
			RemotePort: p.RemotePort,
			LocalIP:    localIP(p.LocalIP),
			LocalPort:  p.LocalPort,
		})
	}

	cli, err := client2.NewClient(&client2.ClientParams{
		ServerAddr: c.Configs.ServerAddr,
		ServerPort: c.Configs.ServerPort,
		Token:      c.Configs.Token,
		Proxies:    proxies,
		Logger:     c.logger,
	})
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %w", err)
	}
	c.cli = cli

	if err := c.cli.Start(); err != nil {
		return fmt.Errorf("启动客户端失败: %w", err)
	}
	c.logger.Infof("客户端启动成功, 连接服务端: %s:%d", c.Configs.ServerAddr, c.Configs.ServerPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	c.logger.Infof("已收到信号 %v, 客户端正在关闭", sig)

	c.cli.Stop()
	c.logger.Infof("客户端已关闭")
	return nil
}

func localIP(ip string) string {
	if ip == "" {
		return configs.DefaultLocalIP
	}
	return ip
}
