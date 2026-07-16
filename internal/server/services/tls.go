package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	"go.uber.org/zap"
)

type Tls struct {
	config *configs.Cert
	logger *zap.Logger
}

func NewTls(config *configs.Cert, logger *zap.Logger) *Tls {
	return &Tls{
		config: config,
		logger: logger,
	}
}

func (t *Tls) Init() (*tls.Config, error) {

	if err := sharedcerts.Ensure(t.config.Dir, t.config.Host); err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(t.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("tls: %w", err)
	}
	caPath := filepath.Join(abs, sharedcerts.FileCACert)
	srvCert := filepath.Join(abs, sharedcerts.FileSrvCert)
	srvKey := filepath.Join(abs, sharedcerts.FileSrvKey)

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Read ca file error: %s", err))
		return nil, fmt.Errorf("failed to read tls CA file")
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		t.logger.Error("Failed to append tls CA file")
		return nil, fmt.Errorf("failed to append tls CA file to cert pool")
	}

	cert, err := tls.LoadX509KeyPair(srvCert, srvKey)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Load tls cert error: %s", err))
		return nil, fmt.Errorf("failed to load server tls certificate")
	}

	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
	}
	return config, nil

}
