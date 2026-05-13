package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	FileCACert  = "ca.crt"
	FileCAKey   = "ca.key"
	FileSrvCert = "server.crt"
	FileSrvKey  = "server.key"
)

func Ensure(dir string, listenHost string) error {

	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("tls: certificate dir: %w", err)
	}
	caCrt := filepath.Join(abs, FileCACert)
	caKey := filepath.Join(abs, FileCAKey)
	srvCrt := filepath.Join(abs, FileSrvCert)
	srvKey := filepath.Join(abs, FileSrvKey)

	exists := func(p string) (bool, error) {
		_, err := os.Stat(p)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	okCAcrt, err := exists(caCrt)
	if err != nil {
		return err
	}
	okCAkey, err := exists(caKey)
	if err != nil {
		return err
	}
	okSrvCrt, err := exists(srvCrt)
	if err != nil {
		return err
	}
	okSrvKey, err := exists(srvKey)
	if err != nil {
		return err
	}

	n := 0
	if okCAcrt {
		n++
	}
	if okCAkey {
		n++
	}
	if okSrvCrt {
		n++
	}
	if okSrvKey {
		n++
	}

	if n == 4 {
		return nil
	}
	if n != 0 {
		return fmt.Errorf(
			"tls: incomplete certificate material in %q (need all of %s, %s, %s, %s or none)",
			abs, FileCACert, FileCAKey, FileSrvCert, FileSrvKey,
		)
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("tls: mkdir %q: %w", abs, err)
	}

	if err := generateAll(abs, listenHost); err != nil {
		return err
	}

	return nil

}

func hostsForSAN(listenHost string) (dns []string, ips []net.IP) {
	dns = []string{"localhost"}
	ips = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	h := strings.TrimSpace(listenHost)
	if h == "" || h == "0.0.0.0" || h == "::" {
		return dns, ips
	}
	if ip := net.ParseIP(h); ip != nil {
		return dns, appendUniqueIP(ips, ip)
	}
	return appendUniqueDNS(dns, h), ips
}

func appendUniqueIP(ips []net.IP, ip net.IP) []net.IP {
	for _, x := range ips {
		if x.Equal(ip) {
			return ips
		}
	}
	return append(ips, ip)
}

func appendUniqueDNS(names []string, name string) []string {
	for _, x := range names {
		if x == name {
			return names
		}
	}
	return append(names, name)
}

func randSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}

func writePEM(path string, blockType string, der []byte, mode os.FileMode) error {
	buf := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	return os.WriteFile(path, buf, mode)
}

func generateAll(dir string, listenHost string) error {

	caPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("tls: generate CA key: %w", err)
	}
	srvPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("tls: generate server key: %w", err)
	}

	caSerial, err := randSerial()
	if err != nil {
		return err
	}
	caTpl := &x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{Organization: []string{"nextunnel"}, CommonName: "nextunnel-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caPriv.PublicKey, caPriv)
	if err != nil {
		return fmt.Errorf("tls: create CA certificate: %w", err)
	}

	srvSerial, err := randSerial()
	if err != nil {
		return err
	}
	dns, ips := hostsForSAN(listenHost)
	srvTpl := &x509.Certificate{
		SerialNumber: srvSerial,
		Subject:      pkix.Name{Organization: []string{"nextunnel"}, CommonName: "nextunnel-server"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
		IPAddresses:  ips,
	}

	srvDER, err := x509.CreateCertificate(rand.Reader, srvTpl, caTpl, &srvPriv.PublicKey, caPriv)
	if err != nil {
		return fmt.Errorf("tls: create server certificate: %w", err)
	}

	caCrt := filepath.Join(dir, FileCACert)
	caKey := filepath.Join(dir, FileCAKey)
	srvCrt := filepath.Join(dir, FileSrvCert)
	srvKey := filepath.Join(dir, FileSrvKey)

	if err := writePEM(caCrt, "CERTIFICATE", caDER, 0o644); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileCACert, err)
	}
	caPrivDER := x509.MarshalPKCS1PrivateKey(caPriv)
	if err := writePEM(caKey, "RSA PRIVATE KEY", caPrivDER, 0o600); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileCAKey, err)
	}
	if err := writePEM(srvCrt, "CERTIFICATE", srvDER, 0o644); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileSrvCert, err)
	}
	srvPrivDER := x509.MarshalPKCS1PrivateKey(srvPriv)
	if err := writePEM(srvKey, "RSA PRIVATE KEY", srvPrivDER, 0o600); err != nil {
		return fmt.Errorf("tls: write %s: %w", FileSrvKey, err)
	}

	return nil

}
