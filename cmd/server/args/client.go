package args

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	services2 "github.com/xiaotiancaipro/nextunnel/internal/server/services"
	"github.com/xiaotiancaipro/nextunnel/internal/server/utils/certs"
	logger_ "github.com/xiaotiancaipro/nextunnel/internal/server/utils/logger"
	"github.com/xiaotiancaipro/nextunnel/internal/server/utils/timezone"
	"gorm.io/gorm"
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

func newClientRegistry(cfg *configs.Configs) (*services2.ClientRegistry, error) {
	db, err := newDB(cfg)
	if err != nil {
		return nil, err
	}
	return services2.NewClientRegistry(db), nil
}

func newClientServices(cfg *configs.Configs) (*services2.ClientRegistry, *services2.ClientCertRegistry, error) {
	db, err := newDB(cfg)
	if err != nil {
		return nil, nil, err
	}
	return services2.NewClientRegistry(db), services2.NewClientCertRegistry(db, cfg.Cert.Dir, cfg.Cert.Host), nil
}

func newDB(cfg *configs.Configs) (*gorm.DB, error) {
	logger, err := logger_.NewLogger(cfg.Logs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	db, err := clients.NewDB(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

func parseExpiresAt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := timezone.ParseRFC3339(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --expires-at value: %w", err)
	}
	return &parsed, nil
}

func formatExpires(expiresAt *time.Time) string {
	if expiresAt == nil {
		return "never"
	}
	return timezone.FormatUTC(*expiresAt)
}

func certOutputDir(cfg *configs.Configs, clientName, certID string) (string, error) {
	recordPath := certs.RelClientCertPath(clientName, certID)
	return certs.AbsCertPath(cfg.Cert.Dir, recordPath)
}

func ensureOutputDir(outDir string) (string, error) {
	outAbs, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("output path: %w", err)
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", outAbs, err)
	}
	return outAbs, nil
}
