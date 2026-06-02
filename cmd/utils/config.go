package utils

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
)

func LoadConfig(cmd *cobra.Command) *configs.Configs {

	flag, err := cmd.Flags().GetString("config")
	if err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	file, err := filepath.Abs(flag)
	if err != nil {
		cmd.PrintErrf("Invalid --config path: %v\n", err)
		os.Exit(1)
	}

	c, err := configs.NewConfigs(file)
	if err != nil {
		cmd.PrintErrf("Failed to load config, %v\n", err)
		os.Exit(1)
	}
	return c

}
