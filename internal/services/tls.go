package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/xiaotiancaipro/nextunnel-client/internal/configs"
	"go.uber.org/zap"
)

type Tls struct {
	Config *configs.Cert
	Logger *zap.Logger
}

func (t *Tls) Init() (*tls.Config, error) {
	caCert, err := os.ReadFile(t.Config.CaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tls ca_file: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append tls ca_file to cert pool")
	}
	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
		RootCAs:            pool,
	}
	if err := t.LoadCertificate(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (t *Tls) LoadCertificate(config *tls.Config) error {
	if t.Config.CertFile == "" || t.Config.KeyFile == "" {
		return fmt.Errorf("tls cert_file and key_file are required")
	}
	cert, err := tls.LoadX509KeyPair(t.Config.CertFile, t.Config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client tls certificate: %w", err)
	}
	config.Certificates = []tls.Certificate{cert}
	return nil
}
