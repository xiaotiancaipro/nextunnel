package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
)

const (
	EnvConfigPath     = "NEXTUNNEL_SERVER_CONFIG"
	DefaultConfigPath = "nextunnel-server.toml"
)

func LoadConfig(cmd *cobra.Command) *configs.Configs {

	path, err := resolveConfigPath(cmd)
	if err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	file, err := filepath.Abs(path)
	if err != nil {
		cmd.PrintErrf("Invalid config path %q: %v\n", path, err)
		os.Exit(1)
	}

	c, err := configs.NewConfigs(file)
	if err != nil {
		cmd.PrintErrf("Failed to load config, %v\n", err)
		os.Exit(1)
	}

	return c

}

func resolveConfigPath(cmd *cobra.Command) (string, error) {
	if isConfigFlagSet(cmd) {
		return cmd.Flags().GetString("config")
	}
	if env := strings.TrimSpace(os.Getenv(EnvConfigPath)); env != "" {
		return env, nil
	}
	return DefaultConfigPath, nil
}

func isConfigFlagSet(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("config"); f != nil && f.Changed {
			return true
		}
	}
	return false
}
