package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	clientconfigs "github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	serverconfigs "github.com/xiaotiancaipro/nextunnel/internal/server/configs"
)

const (
	ServerDefaultConfigPath = "nextunnel-server.toml"
	ServerEnvConfigPath     = "NEXTUNNEL_SERVER_CONFIG"
)

const (
	ClientDefaultConfigPath = "nextunnel-client.toml"
)

type ConfigSpec struct {
	DefaultPath string
	EnvVar      string
}

type configLoader[T any] func(path string) (*T, error)

func LoadClientConfig(cmd *cobra.Command) *clientconfigs.Configs {
	return loadConfig(
		cmd,
		ConfigSpec{DefaultPath: ClientDefaultConfigPath},
		clientconfigs.NewConfigs,
		"Failed to load client config",
	)
}

func LoadServerConfig(cmd *cobra.Command) *serverconfigs.Configs {
	return loadConfig(
		cmd,
		ConfigSpec{
			DefaultPath: ServerDefaultConfigPath,
			EnvVar:      ServerEnvConfigPath,
		},
		serverconfigs.NewConfigs,
		"Failed to load config",
	)
}

func loadConfig[T any](cmd *cobra.Command, spec ConfigSpec, loader configLoader[T], failMsg string) *T {
	path, err := resolveConfigPath(cmd, spec)
	if err != nil {
		cmd.PrintErrf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	file, err := filepath.Abs(path)
	if err != nil {
		cmd.PrintErrf("Invalid config path %q: %v\n", path, err)
		os.Exit(1)
	}

	c, err := loader(file)
	if err != nil {
		cmd.PrintErrf("%s, %v\n", failMsg, err)
		os.Exit(1)
	}
	return c
}

func resolveConfigPath(cmd *cobra.Command, spec ConfigSpec) (string, error) {
	if isConfigFlagSet(cmd) {
		return cmd.Flags().GetString("config")
	}
	if spec.EnvVar != "" {
		if env := strings.TrimSpace(os.Getenv(spec.EnvVar)); env != "" {
			return env, nil
		}
	}
	return spec.DefaultPath, nil
}

func isConfigFlagSet(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("config"); f != nil && f.Changed {
			return true
		}
	}
	return false
}
