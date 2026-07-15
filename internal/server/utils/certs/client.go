package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	FileClientCert     = "client.crt"
	FileClientKey      = "client.key"
	DirClients         = "clients"
	neverExpiresYear   = 2090
	clientNeverExpires = 100 // years from now when no expiry is requested
)

func GenerateClientPEM(tlsDir string, listenHost string, expiresAt *time.Time) (certPEM, keyPEM []byte, err error) {
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
	notAfter, err := ResolveNotAfter(expiresAt)
	if err != nil {
		return nil, nil, err
	}
	clientTpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"nextunnel"}, CommonName: "nextunnel-client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     notAfter,
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

func ResolveNotAfter(expiresAt *time.Time) (time.Time, error) {
	if expiresAt == nil {
		return time.Now().AddDate(clientNeverExpires, 0, 0), nil
	}
	t := expiresAt.UTC()
	if !t.After(time.Now().UTC()) {
		return time.Time{}, fmt.Errorf("tls: certificate expiry must be in the future")
	}
	return t, nil
}

func IsNeverExpires(notAfter time.Time) bool {
	return notAfter.UTC().Year() >= neverExpiresYear
}

func ParseSerial(certPEM []byte) (string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", fmt.Errorf("tls: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("tls: parse certificate: %w", err)
	}
	return cert.SerialNumber.String(), nil
}

func RelClientCertPath(clientName, certID string) string {
	return path.Join(DirClients, clientName, certID)
}

func AbsCertPath(certDir, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", fmt.Errorf("tls: certificate path is empty")
	}
	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("tls: invalid certificate path %q", relPath)
	}
	certAbs, err := filepath.Abs(certDir)
	if err != nil {
		return "", fmt.Errorf("tls: certificate dir: %w", err)
	}
	abs := filepath.Join(certAbs, filepath.FromSlash(relPath))
	if !strings.HasPrefix(abs, certAbs+string(os.PathSeparator)) && abs != certAbs {
		return "", fmt.Errorf("tls: invalid certificate path %q", relPath)
	}
	return abs, nil
}

func ClientCertDir(tlsDir string, clientName string) (string, error) {
	if err := validateClientName(clientName); err != nil {
		return "", err
	}
	tlsAbs, err := filepath.Abs(tlsDir)
	if err != nil {
		return "", fmt.Errorf("tls: certificate dir: %w", err)
	}
	return filepath.Join(tlsAbs, DirClients, strings.TrimSpace(clientName)), nil
}

func ReadCertFiles(dir string) ([]byte, []byte, error) {
	certPEM, err := os.ReadFile(filepath.Join(dir, FileClientCert))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("tls: certificate not found")
		}
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, FileClientKey))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("tls: certificate not found")
		}
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

func RemoveCertDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("tls: remove client cert dir %q: %w", dir, err)
	}
	return nil
}

func RemoveClientCertDir(tlsDir string, clientName string) error {
	outDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return err
	}
	return RemoveCertDir(outDir)
}

func validateClientName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("tls: client name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("tls: invalid client name %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("tls: client name cannot contain path separators")
	}
	return nil
}

func writeClientPEMFiles(outDir string, certPEM, keyPEM []byte) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("tls: mkdir %q: %w", outDir, err)
	}
	clientCrt := filepath.Join(outDir, FileClientCert)
	clientKey := filepath.Join(outDir, FileClientKey)
	if err := os.WriteFile(clientCrt, certPEM, 0o644); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileClientCert, err)
	}
	if err := os.WriteFile(clientKey, keyPEM, 0o600); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileClientKey, err)
	}
	return nil
}

// WriteClientPEMToDir writes certificate and key PEM files into outDir, overwriting existing files.
func WriteClientPEMToDir(outDir string, certPEM, keyPEM []byte) error {
	return writeClientPEMFiles(outDir, certPEM, keyPEM)
}

func GenerateClientToDir(tlsDir string, listenHost string, outDir string, expiresAt *time.Time) error {
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

	certPEM, keyPEM, err := GenerateClientPEM(tlsDir, listenHost, expiresAt)
	if err != nil {
		return err
	}

	return writeClientPEMFiles(outAbs, certPEM, keyPEM)
}
