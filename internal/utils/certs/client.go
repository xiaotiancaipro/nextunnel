package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	FileClientCert = "client.crt"
	FileClientKey  = "client.key"
)

func GenerateClientPEM(tlsDir string, listenHost string) (certPEM, keyPEM []byte, err error) {
	if err = Ensure(tlsDir, listenHost); err != nil {
		return nil, nil, err
	}
	tlsAbs, err := filepath.Abs(tlsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("tls: %w", err)
	}

	caCertPEM, err := os.ReadFile(filepath.Join(tlsAbs, FileCACert))
	if err != nil {
		return nil, nil, fmt.Errorf("tls: read CA cert: %w", err)
	}
	caKeyPEM, err := os.ReadFile(filepath.Join(tlsAbs, fileCAKey))
	if err != nil {
		return nil, nil, fmt.Errorf("tls: read CA key: %w", err)
	}

	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		return nil, nil, fmt.Errorf("tls: invalid CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("tls: parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("tls: invalid CA key PEM")
	}
	caPriv, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("tls: parse CA private key: %w", err)
	}

	clientPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("tls: generate client key: %w", err)
	}

	serial, err := randSerial()
	if err != nil {
		return nil, nil, err
	}
	dns, ips := hostsForSAN(listenHost)
	clientTpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"nextunnel"}, CommonName: "nextunnel-client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:     dns,
		IPAddresses:  ips,
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTpl, caCert, &clientPriv.PublicKey, caPriv)
	if err != nil {
		return nil, nil, fmt.Errorf("tls: create client certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	privDER := x509.MarshalPKCS1PrivateKey(clientPriv)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	return certPEM, keyPEM, nil
}

func GenerateClientToDir(tlsDir string, listenHost string, outDir string) error {

	if outDir == "" {
		return fmt.Errorf("tls: client cert output directory is empty")
	}
	outAbs, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("tls: client output path: %w", err)
	}

	clientCrt := filepath.Join(outAbs, FileClientCert)
	clientKey := filepath.Join(outAbs, FileClientKey)
	if _, err := os.Stat(clientCrt); err == nil {
		return fmt.Errorf("tls: %s already exists", clientCrt)
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(clientKey); err == nil {
		return fmt.Errorf("tls: %s already exists", clientKey)
	} else if !os.IsNotExist(err) {
		return err
	}

	certPEM, keyPEM, err := GenerateClientPEM(tlsDir, listenHost)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return fmt.Errorf("tls: mkdir %q: %w", outAbs, err)
	}
	if err := os.WriteFile(clientCrt, certPEM, 0o644); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileClientCert, err)
	}
	if err := os.WriteFile(clientKey, keyPEM, 0o600); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileClientKey, err)
	}

	return nil

}
