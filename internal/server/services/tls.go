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
	Config *configs.Cert
	Logger *zap.Logger
}

func (s *Tls) Init() (*tls.Config, error) {

	if err := sharedcerts.Ensure(s.Config.Dir, s.Config.Host); err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(s.Config.Dir)
	if err != nil {
		return nil, fmt.Errorf("tls: %w", err)
	}
	caPath := filepath.Join(abs, sharedcerts.FileCACert)
	srvCert := filepath.Join(abs, sharedcerts.FileSrvCert)
	srvKey := filepath.Join(abs, sharedcerts.FileSrvKey)

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to read tls ca file: %v", err))
		return nil, fmt.Errorf("failed to read tls CA file")
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		s.Logger.Error("failed to append tls ca file to cert pool")
		return nil, fmt.Errorf("failed to append tls CA file to cert pool")
	}

	cert, err := tls.LoadX509KeyPair(srvCert, srvKey)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("failed to load server tls certificate: %v", err))
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
