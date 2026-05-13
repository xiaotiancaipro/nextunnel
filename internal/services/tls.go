package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/certs"
	"go.uber.org/zap"
)

type Tls struct {
	Config     *configs.Tls
	ServerAddr string
	Logger     *zap.Logger
}

func (t *Tls) Init() (*tls.Config, error) {

	if err := certs.Ensure(t.Config.Dir, t.ServerAddr); err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(t.Config.Dir)
	if err != nil {
		return nil, fmt.Errorf("tls: %w", err)
	}
	caPath := filepath.Join(abs, certs.FileCACert)
	srvCert := filepath.Join(abs, certs.FileSrvCert)
	srvKey := filepath.Join(abs, certs.FileSrvKey)

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		t.Logger.Error(fmt.Sprintf("Read ca file error: %s", err))
		return nil, fmt.Errorf("failed to read tls CA file")
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		t.Logger.Error("Failed to append tls CA file")
		return nil, fmt.Errorf("failed to append tls CA file to cert pool")
	}

	cert, err := tls.LoadX509KeyPair(srvCert, srvKey)
	if err != nil {
		t.Logger.Error(fmt.Sprintf("Load tls cert error: %s", err))
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
