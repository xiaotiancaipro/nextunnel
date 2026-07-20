package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	"go.uber.org/zap"
)

type Tls struct {
	Config *configs.Cert
	Logger *zap.Logger
}

func (s *Tls) Init() (*tls.Config, error) {
	caCert, err := os.ReadFile(s.Config.CaFile)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to read tls ca_file: %v", err))
		return nil, fmt.Errorf("failed to read tls ca_file")
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		s.Logger.Error("failed to append tls ca_file to cert pool")
		return nil, fmt.Errorf("failed to append tls ca_file to cert pool")
	}
	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
		RootCAs:            pool,
	}
	if err := s.LoadCertificate(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (s *Tls) LoadCertificate(config *tls.Config) error {
	if s.Config.CertFile == "" || s.Config.KeyFile == "" {
		s.Logger.Error("tls cert_file and key_file are required")
		return fmt.Errorf("tls cert_file and key_file are required")
	}
	cert, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to load client tls certificate: %v", err))
		return fmt.Errorf("failed to load client tls certificate")
	}
	config.Certificates = []tls.Certificate{cert}
	return nil
}
