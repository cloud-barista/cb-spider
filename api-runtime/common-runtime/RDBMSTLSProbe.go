// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, September 2026.

package commonruntime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// RDBMSCACertInfo describes the top-of-chain certificate captured from a live TLS handshake
// against the RDBMS endpoint — independent of the SQL connection used for the other secure
// transport checks. It's captured live (rather than sourced from each CSP's own API/docs)
// because that source is inconsistent across CSPs: some expose it via API (GCP's serverCaCert,
// IBM's connection certificate_base64), some only via a static documented download (AWS, Azure),
// and several (Alibaba, Tencent, NCP, NHN, OpenStack) have no documented programmatic source.
type RDBMSCACertInfo struct {
	// PEM is the certificate in PEM format — usable directly as a client's ssl-ca / sslrootcert file.
	// Caveat: this is the TOP-MOST certificate the server presented during the handshake, which is
	// often an intermediate CA rather than the ultimate self-signed root (well-behaved servers don't
	// send the root — clients are expected to already trust it independently). In practice this is
	// what most "grab the CA off the server" workflows use, but a CSP-published root, when one is
	// documented, is more authoritative.
	PEM      string `json:"PEM"`
	Subject  string `json:"Subject"`
	Issuer   string `json:"Issuer"`
	NotAfter string `json:"NotAfter"` // RFC3339

	// IsSelfSigned is true only when this certificate's signature cryptographically verifies
	// against its own public key (i.e. it's an actual root CA), not merely Subject == Issuer.
	// false means it's an intermediate CA — still usable as ssl-ca/sslrootcert (the server will
	// keep presenting it in the chain), but it isn't the ultimate trust anchor.
	IsSelfSigned bool `json:"IsSelfSigned"`
}

// fetchRDBMSCACertificate captures the server's top-of-chain certificate via a live TLS
// handshake and converts it into RDBMSCACertInfo. Returns (nil, err) if the probe fails for any
// reason (server offers no TLS, network unreachable, unsupported engine, ...) — callers should
// treat this as best-effort and not fail the overall request on error.
func fetchRDBMSCACertificate(engine, host, port string) (*RDBMSCACertInfo, error) {
	certs, err := fetchRDBMSServerCertChain(engine, host, port)
	if err != nil {
		return nil, err
	}
	top := certs[len(certs)-1]
	return &RDBMSCACertInfo{
		PEM:          string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: top.Raw})),
		Subject:      top.Subject.String(),
		Issuer:       top.Issuer.String(),
		NotAfter:     top.NotAfter.Format(time.RFC3339),
		IsSelfSigned: top.CheckSignatureFrom(top) == nil,
	}, nil
}

// fetchRDBMSServerCertChain opens a bare TCP connection to host:port and performs just enough of
// the engine's wire protocol to trigger a TLS upgrade (MySQL's SSLRequest packet, or PostgreSQL's
// SSLRequest message), then captures the certificate chain the server presents. No authentication
// happens — the connection is closed right after the TLS handshake completes.
//
// Retries once on failure: in practice a hung/dropped probe connection (e.g. one packet lost on
// the path) is much more effectively worked around by opening a brand new TCP connection (new
// source port, new route) than by waiting longer on the same stuck one. A second attempt that's
// actually going to work typically succeeds quickly, so this doesn't meaningfully slow the
// common case; it only adds latency when the first attempt was already failing anyway.
func fetchRDBMSServerCertChain(engine, host, port string) ([]*x509.Certificate, error) {
	lowerEngine := strings.ToLower(engine)
	var probe func(string, string) ([]*x509.Certificate, error)
	switch {
	case strings.Contains(lowerEngine, "mysql") || strings.Contains(lowerEngine, "mariadb"):
		probe = fetchMySQLServerCertChain
	case strings.Contains(lowerEngine, "postgres"):
		probe = fetchPostgresServerCertChain
	default:
		return nil, fmt.Errorf("unsupported engine for TLS cert probe: %s", engine)
	}

	const maxAttempts = 2
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		certs, err := probe(host, port)
		if err == nil {
			return certs, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("after %d attempts, last error: %w", maxAttempts, lastErr)
}

// fetchMySQLServerCertChain speaks just enough MySQL wire protocol to trigger a TLS upgrade:
// read the server's initial handshake packet, send an SSLRequest packet (a Handshake Response
// Packet truncated to its fixed-size header, with the CLIENT_SSL flag set), then hand the raw
// connection to tls.Client. No username/password is ever sent.
func fetchMySQLServerCertChain(host, port string) ([]*x509.Certificate, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 8*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read (and discard) the server's initial handshake packet.
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, fmt.Errorf("read initial handshake header: %w", err)
	}
	length := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("read initial handshake payload: %w", err)
	}
	if len(payload) > 0 && payload[0] == 0xff {
		return nil, fmt.Errorf("server returned an error on the initial handshake")
	}

	// SSLRequest packet: the fixed-size prefix of a Handshake Response Packet (protocol 41),
	// with CLIENT_SSL set, sent BEFORE the TLS upgrade instead of a full auth response.
	//
	// The flag set below mirrors exactly what go-sql-driver/mysql itself sends by default
	// (mysqlConn.initCapabilities, minus the flags it only sets conditionally on config we don't
	// use here). The MySQL/MariaDB protocol spec says a minimal CLIENT_SSL|CLIENT_PROTOCOL_41-only
	// packet should be enough, but a prior version of this probe that only set those two (plus a
	// couple of others) reliably hung waiting for a response from a MySQL 8.x RDS instance while
	// succeeding against a MariaDB instance on the same account/VPC/security group — i.e. some
	// engine build is pickier about the capability flags than the spec implies. Sending the exact
	// flags a real client sends avoids depending on that undocumented leniency.
	const (
		clientMySQL                      = 0x00000001
		clientLongFlag                   = 0x00000004
		clientProtocol41                 = 0x00000200
		clientSSL                        = 0x00000800
		clientTransactions               = 0x00002000
		clientSecureConn                 = 0x00008000
		clientMultiResults               = 0x00020000
		clientPluginAuth                 = 0x00080000
		clientConnectAttrs               = 0x00100000
		clientPluginAuthLenEncClientData = 0x00200000
		clientLocalFiles                 = 0x00000080
		clientDeprecateEOF               = 0x01000000
	)
	capabilityFlags := uint32(clientMySQL | clientLongFlag | clientProtocol41 | clientSSL |
		clientTransactions | clientSecureConn | clientMultiResults | clientPluginAuth |
		clientConnectAttrs | clientPluginAuthLenEncClientData | clientLocalFiles | clientDeprecateEOF)

	sslRequest := make([]byte, 32)
	binary.LittleEndian.PutUint32(sslRequest[0:4], capabilityFlags)
	binary.LittleEndian.PutUint32(sslRequest[4:8], 0) // max_packet_size: unspecified, matching the real driver
	sslRequest[8] = 0x2d                              // character_set: utf8mb4_general_ci (irrelevant here)
	// bytes 9..31 are the reserved filler, left zeroed

	packet := make([]byte, 4+len(sslRequest))
	packet[0] = byte(len(sslRequest))
	packet[1] = byte(len(sslRequest) >> 8)
	packet[2] = byte(len(sslRequest) >> 16)
	packet[3] = 1 // sequence id: server's initial handshake used 0, this response is 1
	copy(packet[4:], sslRequest)

	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("write SSLRequest packet: %w", err)
	}

	tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
	if err := tlsConn.HandshakeContext(context.Background()); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("server presented no certificate")
	}
	return certs, nil
}

// fetchPostgresServerCertChain speaks just enough of the PostgreSQL frontend/backend protocol to
// trigger a TLS upgrade: send an SSLRequest message, expect a single 'S' byte back, then hand the
// raw connection to tls.Client. No startup message (database/user) is ever sent.
func fetchPostgresServerCertChain(host, port string) ([]*x509.Certificate, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 8*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// SSLRequest message: length(int32)=8, followed by the fixed SSL request code 80877103.
	req := make([]byte, 8)
	binary.BigEndian.PutUint32(req[0:4], 8)
	binary.BigEndian.PutUint32(req[4:8], 80877103)
	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("write SSLRequest message: %w", err)
	}

	resp := make([]byte, 1)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, fmt.Errorf("read SSLRequest response: %w", err)
	}
	if resp[0] != 'S' {
		return nil, fmt.Errorf("server does not support SSL")
	}

	tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
	if err := tlsConn.HandshakeContext(context.Background()); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("server presented no certificate")
	}
	return certs, nil
}
