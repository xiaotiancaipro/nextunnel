package certs

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
)

func ClientLeafCertSHA256(conn net.Conn) ([sha256.Size]byte, error) {
	var z [sha256.Size]byte
	tc, ok := conn.(*tls.Conn)
	if !ok {
		return z, fmt.Errorf("not a TLS connection")
	}
	state := tc.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return z, fmt.Errorf("no peer certificate")
	}
	return sha256.Sum256(state.PeerCertificates[0].Raw), nil
}

func CertPEMSHA256(certPEM []byte) ([sha256.Size]byte, error) {
	var z [sha256.Size]byte
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return z, fmt.Errorf("tls: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return z, fmt.Errorf("tls: parse certificate: %w", err)
	}
	return sha256.Sum256(cert.Raw), nil
}
