package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

func ParseExpiresAt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := sharedtimezone.ParseRFC3339(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --expires-at value: %w", err)
	}
	return &parsed, nil
}

func FormatExpires(expiresAt *time.Time) string {
	if expiresAt == nil {
		return "never"
	}
	return sharedtimezone.FormatUTC(*expiresAt)
}

func CertOutputDir(cfg *configs.Configs, clientName, certID string) (string, error) {
	recordPath := sharedcerts.RelClientCertPath(clientName, certID)
	return sharedcerts.AbsCertPath(cfg.Cert.Dir, recordPath)
}

func EnsureOutputDir(outDir string) (string, error) {
	outAbs, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("output path: %w", err)
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", outAbs, err)
	}
	return outAbs, nil
}
