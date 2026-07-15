package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/services"
	"github.com/xiaotiancaipro/nextunnel/internal/utils/certs"
	"github.com/xiaotiancaipro/nextunnel/internal/utils/timezone"
)

func ListClientCerts(cmd *cobra.Command, cfg *configs.Configs, clientName string) error {
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

	items, err := certService.List(client.Id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "no certificates for client %q\n", clientName)
		return nil
	}

	for _, item := range items {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tcreated=%s\texpires=%s\tserial=%s\n",
			item.ID,
			timezone.FormatUTC(item.CreatedAt),
			formatExpires(item.ExpiresAt),
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

	info, err := certService.Create(client, expiresAt)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created certificate %q for client %q (expires=%s, serial=%s)\n",
		info.ID, clientName, formatExpires(info.ExpiresAt), info.Serial)
	return nil
}

func DeleteClientCert(cmd *cobra.Command, cfg *configs.Configs, clientName, certIDRaw string) error {
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
	certID, err := services.ParseCertID(certIDRaw)
	if err != nil {
		return err
	}

	if err := certService.Delete(client.Id, certID); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deleted certificate %q for client %q\n", certID, clientName)
	return nil
}

func DownloadClientCert(cmd *cobra.Command, cfg *configs.Configs, clientName, certIDRaw, outDir string) error {
	clientName = strings.TrimSpace(clientName)
	outDir = strings.TrimSpace(outDir)
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
	certID, err := services.ParseCertID(certIDRaw)
	if err != nil {
		return err
	}

	certPEM, keyPEM, err := certService.ReadFiles(client.Id, certID)
	if err != nil {
		return err
	}

	if outDir == "" {
		outDir, err = certOutputDir(cfg, clientName, certID.String())
		if err != nil {
			return err
		}
	} else {
		outDir, err = ensureOutputDir(outDir)
		if err != nil {
			return err
		}
	}

	if err := certs.WriteClientPEMToDir(outDir, certPEM, keyPEM); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s and %s for certificate %q in %s\n",
		certs.FileClientCert, certs.FileClientKey, certID, outDir)
	return nil
}
