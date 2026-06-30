package args

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/services"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
	logger_ "github.com/xiaotiancaipro/nextunnel-server/internal/utils/logger"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/timeformat"
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

func GenerateCerts(cmd *cobra.Command, cfg *configs.Configs, out, clientName, expiresAtRaw string) error {
	out = strings.TrimSpace(out)
	clientName = strings.TrimSpace(clientName)
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}

	clientService, certService, err := newClientServices(cfg)
	if err != nil {
		return err
	}
	client, err := clientService.GetByName(clientName)
	if err != nil {
		return err
	}

	expiresAt, err := parseExpiresAt(expiresAtRaw)
	if err != nil {
		return err
	}

	if out == "" {
		info, err := certService.Create(client, expiresAt)
		if err != nil {
			return err
		}
		abs, err := certs.AbsCertPath(cfg.Cert.Dir, certs.RelClientCertPath(clientName, info.ID))
		if err != nil {
			return err
		}
		expires := formatExpires(info.ExpiresAt)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created certificate %q (expires=%s) for client %q in %s\n",
			info.ID, expires, clientName, abs)
		return nil
	}

	if err := certs.GenerateClientToDir(cfg.Cert.Dir, cfg.Cert.Host, out, expiresAt); err != nil {
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
	db, err := newDB(cfg)
	if err != nil {
		return nil, err
	}
	return services.NewClientRegistry(db), nil
}

func newClientCertRegistry(cfg *configs.Configs) (*services.ClientCertRegistry, error) {
	db, err := newDB(cfg)
	if err != nil {
		return nil, err
	}
	return services.NewClientCertRegistry(db, cfg.Cert.Dir, cfg.Cert.Host), nil
}

func newClientServices(cfg *configs.Configs) (*services.ClientRegistry, *services.ClientCertRegistry, error) {
	db, err := newDB(cfg)
	if err != nil {
		return nil, nil, err
	}
	return services.NewClientRegistry(db), services.NewClientCertRegistry(db, cfg.Cert.Dir, cfg.Cert.Host), nil
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
	parsed, err := timeformat.ParseRFC3339(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --expires-at value: %w", err)
	}
	return &parsed, nil
}

func formatExpires(expiresAt *time.Time) string {
	if expiresAt == nil {
		return "never"
	}
	return timeformat.FormatUTC(*expiresAt)
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
