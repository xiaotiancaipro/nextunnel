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
	config *configs.Tls
	logger *zap.Logger
}

func NewTls(config *configs.Configs, logger *zap.Logger) *Tls {
	return &Tls{
		config: config.Tls,
		logger: logger,
	}
}

func (t *Tls) Init() (*tls.Config, error) {
	caCert, err := os.ReadFile(t.config.CaFile)
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
		ServerName:         t.config.ServerName,
		RootCAs:            pool,
	}
	if err := t.LoadCertificate(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (t *Tls) LoadCertificate(config *tls.Config) error {
	if t.config.CertFile == "" || t.config.KeyFile == "" {
		return fmt.Errorf("tls cert_file and key_file are required")
	}
	cert, err := tls.LoadX509KeyPair(t.config.CertFile, t.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client tls certificate: %w", err)
	}
	config.Certificates = []tls.Certificate{cert}
	return nil
}
