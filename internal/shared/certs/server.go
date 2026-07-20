package certs

import (
	"crypto/sha256"
	"crypto/tls"
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
