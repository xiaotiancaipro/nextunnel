package root

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/apps"
	configs_ "github.com/xiaotiancaipro/nextunnel/internal/configs"
)

func NewServer() *cobra.Command {
	fc := func(cmd *cobra.Command, _ []string) {
		configFile, err1 := cmd.Flags().GetString("config")
		configs, err2 := configs_.NewServer(configFile)
		if err := errors.Join(err1, err2); err != nil {
			cmd.PrintErrf("加载服务端配置失败, %v\n", err)
			os.Exit(1)
		}
		app := &apps.Server{Configs: configs}
		if err := app.StartAndStop(); err != nil {
			cmd.PrintErrf("服务端异常, %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	c := &cobra.Command{
		Use:   "server",
		Short: "启动内网穿透服务端 (server)",
		Args:  cobra.ExactArgs(0),
		Run:   fc,
	}
	c.Flags().StringP("config", "c", "config/nextunnel-server.toml", "服务端配置文件路径")
	return c
}
