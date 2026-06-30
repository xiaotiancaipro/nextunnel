package args

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
)

func ListClientCerts(cmd *cobra.Command, cfg *configs.Configs, clientName string) error {
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

	items, err := certs.ListClientCerts(cfg.Cert.Dir, clientName)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "no certificates for client %q\n", clientName)
		return nil
	}

	for _, item := range items {
		expires := "never"
		if item.ExpiresAt != nil {
			expires = item.ExpiresAt.UTC().Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tcreated=%s\texpires=%s\tserial=%s\n",
			item.ID,
			item.CreatedAt.UTC().Format(time.RFC3339),
			expires,
			item.Serial,
		)
	}
	return nil
}

func CreateClientCert(cmd *cobra.Command, cfg *configs.Configs, clientName, expiresAtRaw string) error {
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

	var expiresAt *time.Time
	expiresAtRaw = strings.TrimSpace(expiresAtRaw)
	if expiresAtRaw != "" {
		parsed, err := time.Parse(time.RFC3339, expiresAtRaw)
		if err != nil {
			return fmt.Errorf("invalid --expires-at value: %w", err)
		}
		expiresAt = &parsed
	}

	info, _, _, err := certs.CreateClientCert(cfg.Cert.Dir, cfg.Cert.Host, clientName, expiresAt)
	if err != nil {
		return err
	}

	expires := "never"
	if info.ExpiresAt != nil {
		expires = info.ExpiresAt.UTC().Format(time.RFC3339)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created certificate %q for client %q (expires=%s, serial=%s)\n",
		info.ID, clientName, expires, info.Serial)
	return nil
}

func DeleteClientCert(cmd *cobra.Command, cfg *configs.Configs, clientName, certID string) error {
	clientName = strings.TrimSpace(clientName)
	certID = strings.TrimSpace(certID)
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}
	if certID == "" {
		return fmt.Errorf("certificate id is required")
	}

	service, err := newClientRegistry(cfg)
	if err != nil {
		return err
	}
	if _, err := service.GetByName(clientName); err != nil {
		return err
	}

	if err := certs.DeleteClientCert(cfg.Cert.Dir, clientName, certID); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)
	return nil
}

func DownloadClientCert(cmd *cobra.Command, cfg *configs.Configs, clientName, certID, outDir string) error {
	clientName = strings.TrimSpace(clientName)
	certID = strings.TrimSpace(certID)
	outDir = strings.TrimSpace(outDir)
	if clientName == "" {
		return fmt.Errorf("client name is required")
	}
	if certID == "" {
		return fmt.Errorf("certificate id is required")
	}

	service, err := newClientRegistry(cfg)
	if err != nil {
		return err
	}
	if _, err := service.GetByName(clientName); err != nil {
		return err
	}

	certPEM, keyPEM, err := certs.ReadClientCertFiles(cfg.Cert.Dir, clientName, certID)
	if err != nil {
		return err
	}

	if outDir == "" {
		clientDir, err := certs.ClientCertDir(cfg.Cert.Dir, clientName)
		if err != nil {
			return err
		}
		if certID == certs.LegacyCertID {
			outDir = clientDir
		} else {
			outDir = filepath.Join(clientDir, certID)
		}
	} else {
		outAbs, err := filepath.Abs(outDir)
		if err != nil {
			return fmt.Errorf("output path: %w", err)
		}
		outDir = outAbs
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", outDir, err)
		}
	}

	if err := certs.WriteClientPEMToDir(outDir, certPEM, keyPEM); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s and %s for certificate %q in %s\n",
		certs.FileClientCert, certs.FileClientKey, certID, outDir)
	return nil
}
