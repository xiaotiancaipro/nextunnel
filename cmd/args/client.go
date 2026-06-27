package args

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
)

func CreateClient(cmd *cobra.Command, cfg *configs.Configs, name string, portStart, portEnd int) error {
	service, err := newClientRegistry(cfg)
	if err != nil {
		return err
	}

	client, err := service.Create(name, portStart, portEnd)
	if err != nil {
		return err
	}

	if portStart > 0 && portEnd > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=%d-%d)\n", client.Name, client.Id, portStart, portEnd)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created client %q (id=%s, ports=all)\n", client.Name, client.Id)
	}
	return nil
}

func GenerateCerts(cmd *cobra.Command, cfg *configs.Configs, out, clientName string) error {
	out = strings.TrimSpace(out)
	if out == "" {
		return fmt.Errorf("--dir is required")
	}
	clientName = strings.TrimSpace(clientName)
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}

	service, err := newClientRegistry(cfg)
	if err != nil {
		return err
	}
	if _, err := service.GetByName(clientName); err != nil {
		return err
	}

	if err := certs.GenerateClientToDir(cfg.Cert.Dir, cfg.Cert.Host, out); err != nil {
		return err
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s and %s for client %q in %s\n", certs.FileClientCert, certs.FileClientKey, clientName, abs)
	return nil
}

func newClientRegistry(cfg *configs.Configs) (*services.ClientRegistry, error) {
	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return services.NewClientRegistry(db), nil
}
