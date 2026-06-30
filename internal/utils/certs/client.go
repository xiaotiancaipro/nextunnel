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
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	FileClientCert     = "client.crt"
	FileClientKey      = "client.key"
	DirClients         = "clients"
	LegacyCertID       = "default"
	neverExpiresYear   = 2090
	clientNeverExpires = 100 // years from now when no expiry is requested
)

// ClientCertInfo describes a stored client certificate.
type ClientCertInfo struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt *time.Time // nil means never expires
	Serial    string
}

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
	notAfter, err := resolveNotAfter(expiresAt)
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

func resolveNotAfter(expiresAt *time.Time) (time.Time, error) {
	if expiresAt == nil {
		return time.Now().AddDate(clientNeverExpires, 0, 0), nil
	}
	t := expiresAt.UTC()
	if !t.After(time.Now()) {
		return time.Time{}, fmt.Errorf("tls: certificate expiry must be in the future")
	}
	return t, nil
}

func isNeverExpires(notAfter time.Time) bool {
	return notAfter.UTC().Year() >= neverExpiresYear
}

func certInfoFromPEM(id string, certPEM []byte, createdAt time.Time) (ClientCertInfo, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return ClientCertInfo{}, fmt.Errorf("tls: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ClientCertInfo{}, fmt.Errorf("tls: parse certificate: %w", err)
	}
	info := ClientCertInfo{
		ID:        id,
		CreatedAt: createdAt,
		Serial:    cert.SerialNumber.String(),
	}
	if !isNeverExpires(cert.NotAfter) {
		expires := cert.NotAfter.UTC()
		info.ExpiresAt = &expires
	}
	return info, nil
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

func clientCertEntryDir(clientDir, certID string) (string, error) {
	certID = strings.TrimSpace(certID)
	if certID == "" {
		return "", fmt.Errorf("tls: certificate id is required")
	}
	if certID == "." || certID == ".." || strings.ContainsAny(certID, `/\`) {
		return "", fmt.Errorf("tls: invalid certificate id %q", certID)
	}
	return filepath.Join(clientDir, certID), nil
}

func ListClientCerts(tlsDir, clientName string) ([]ClientCertInfo, error) {
	clientDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(clientDir); os.IsNotExist(err) {
		return []ClientCertInfo{}, nil
	} else if err != nil {
		return nil, err
	}

	var items []ClientCertInfo

	legacyCert := filepath.Join(clientDir, FileClientCert)
	if st, err := os.Stat(legacyCert); err == nil && !st.IsDir() {
		certPEM, err := os.ReadFile(legacyCert)
		if err != nil {
			return nil, fmt.Errorf("tls: read %s: %w", FileClientCert, err)
		}
		info, err := certInfoFromPEM(LegacyCertID, certPEM, st.ModTime())
		if err != nil {
			return nil, err
		}
		items = append(items, info)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	entries, err := os.ReadDir(clientDir)
	if err != nil {
		return nil, fmt.Errorf("tls: read client cert dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		certPath := filepath.Join(clientDir, entry.Name(), FileClientCert)
		st, err := os.Stat(certPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("tls: read %s: %w", certPath, err)
		}
		info, err := certInfoFromPEM(entry.Name(), certPEM, st.ModTime())
		if err != nil {
			return nil, err
		}
		items = append(items, info)
	}
	return items, nil
}

func CreateClientCert(tlsDir, listenHost, clientName string, expiresAt *time.Time) (ClientCertInfo, []byte, []byte, error) {
	clientDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return ClientCertInfo{}, nil, nil, err
	}

	certID := uuid.NewString()
	entryDir, err := clientCertEntryDir(clientDir, certID)
	if err != nil {
		return ClientCertInfo{}, nil, nil, err
	}

	certPEM, keyPEM, err := GenerateClientPEM(tlsDir, listenHost, expiresAt)
	if err != nil {
		return ClientCertInfo{}, nil, nil, err
	}
	if err := writeClientPEMFiles(entryDir, certPEM, keyPEM); err != nil {
		return ClientCertInfo{}, nil, nil, err
	}

	info, err := certInfoFromPEM(certID, certPEM, time.Now().UTC())
	if err != nil {
		return ClientCertInfo{}, nil, nil, err
	}
	return info, certPEM, keyPEM, nil
}

func ReadClientCertFiles(tlsDir, clientName, certID string) ([]byte, []byte, error) {
	clientDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return nil, nil, err
	}

	var entryDir string
	if certID == LegacyCertID {
		entryDir = clientDir
	} else {
		entryDir, err = clientCertEntryDir(clientDir, certID)
		if err != nil {
			return nil, nil, err
		}
	}

	certPEM, err := os.ReadFile(filepath.Join(entryDir, FileClientCert))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("tls: certificate %q not found", certID)
		}
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(filepath.Join(entryDir, FileClientKey))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("tls: certificate %q not found", certID)
		}
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

func DeleteClientCert(tlsDir, clientName, certID string) error {
	clientDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return err
	}

	if certID == LegacyCertID {
		certPath := filepath.Join(clientDir, FileClientCert)
		keyPath := filepath.Join(clientDir, FileClientKey)
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			return fmt.Errorf("tls: certificate %q not found", certID)
		} else if err != nil {
			return err
		}
		if err := os.Remove(certPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("tls: remove %s: %w", FileClientCert, err)
		}
		if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("tls: remove %s: %w", FileClientKey, err)
		}
		return nil
	}

	entryDir, err := clientCertEntryDir(clientDir, certID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(entryDir); os.IsNotExist(err) {
		return fmt.Errorf("tls: certificate %q not found", certID)
	} else if err != nil {
		return err
	}
	if err := os.RemoveAll(entryDir); err != nil {
		return fmt.Errorf("tls: remove client cert dir %q: %w", entryDir, err)
	}
	return nil
}

func WriteClientCertDir(tlsDir string, listenHost string, clientName string) (certPEM, keyPEM []byte, err error) {
	info, certPEM, keyPEM, err := CreateClientCert(tlsDir, listenHost, clientName, nil)
	if err != nil {
		return nil, nil, err
	}
	_ = info
	return certPEM, keyPEM, nil
}

func RemoveClientCertDir(tlsDir string, clientName string) error {
	outDir, err := ClientCertDir(tlsDir, clientName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("tls: remove client cert dir %q: %w", outDir, err)
	}
	return nil
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
