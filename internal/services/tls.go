package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"go.uber.org/zap"
)

type Tls struct {
	Config *configs.Tls
	Logger *zap.Logger
}

func (t *Tls) Init() (*tls.Config, error) {
	caCert, err := os.ReadFile(t.Config.CaFile)
	if err != nil {
		t.Logger.Error(fmt.Sprintf("Read ca file error: %s", err))
		return nil, fmt.Errorf("failed to read tls ca_file")
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		t.Logger.Error("Failed to append tls ca_file")
		return nil, fmt.Errorf("failed to append tls ca_file to cert pool")
	}
	cert, err := tls.LoadX509KeyPair(t.Config.CertFile, t.Config.KeyFile)
	if err != nil {
		t.Logger.Error(fmt.Sprintf("Load tls cert error: %s", err))
		return nil, fmt.Errorf("failed to load client tls certificate")
	}
	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
	}
	return config, nil
}
