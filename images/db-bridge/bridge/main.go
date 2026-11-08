// db-bridge: mTLS Postgres bridge.
//
// Listens on 127.0.0.1:5432 (plain TCP). For each app connection it opens
// a connection to the upstream Postgres, performs the Postgres SSL
// negotiation (SSLRequest / 'S'), upgrades that connection to TLS
// presenting the pod's SPIFFE SVID as the client certificate, then pipes
// bytes both ways.
//
// The app speaks the plain Postgres protocol to the bridge (no password);
// the server authenticates the SVID (CN=app) via pg_hba "cert". The SVID
// files are re-read on every new connection, so spiffe-helper rotation is
// absorbed without a restart: existing sessions keep their (still valid)
// old certificate, new connections pick up the renewed one.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	listenAddr = "127.0.0.1:5432"
	upstream   = "postgres-rw.database.svc:5432"
	svidCert   = "/spiffe/svid.crt"
	svidKey    = "/spiffe/svid.key"
	caFile     = "/cnpg-ca/ca.crt"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	// db-bridge -health: liveness probe (is the listener up?)
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		c, err := net.Dial("tcp", listenAddr)
		if err != nil {
			log.Printf("health: %v", err)
			os.Exit(1)
		}
		_ = c.Close()
		return
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("db-bridge listening on %s -> %s", listenAddr, upstream)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
		<-ch
		_ = ln.Close()
	}()

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			return
		}
		go handle(client)
	}
}

// Postgres pre-startup request codes (sent by the client in an 8-byte
// message before its StartupMessage).
const (
	codeSSLRequest    = 80877103
	codeGSSENCRequest = 80877104
)

// readStartup answers the client's own SSLRequest/GSSENCRequest with 'N'
// (loopback stays plaintext) and returns the first real startup-phase
// message (StartupMessage or CancelRequest), to be replayed upstream inside
// the TLS session. Without this, the client's SSLRequest would be forwarded
// into the already-encrypted upstream session and Postgres would answer 'E'.
func readStartup(client net.Conn) ([]byte, error) {
	for {
		var hdr [8]byte
		if _, err := io.ReadFull(client, hdr[:]); err != nil {
			return nil, err
		}
		length := binary.BigEndian.Uint32(hdr[0:4])
		code := binary.BigEndian.Uint32(hdr[4:8])
		if length == 8 && (code == codeSSLRequest || code == codeGSSENCRequest) {
			if _, err := client.Write([]byte{'N'}); err != nil {
				return nil, err
			}
			continue
		}
		if length < 8 || length > 10000 {
			return nil, fmt.Errorf("invalid startup message length %d", length)
		}
		msg := make([]byte, length)
		copy(msg, hdr[:])
		if _, err := io.ReadFull(client, msg[8:]); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

func handle(client net.Conn) {
	defer client.Close()

	startup, err := readStartup(client)
	if err != nil {
		if err != io.EOF { // EOF: the -health probe connects and closes
			log.Printf("client startup: %v", err)
		}
		return
	}

	server, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("dial upstream: %v", err)
		return
	}
	defer server.Close()

	// Postgres SSLRequest: 8-byte message, code 80877103 (0x04D2162F).
	sslRequest := []byte{0, 0, 0, 8, 0x04, 0xD2, 0x16, 0x2F}
	if _, err := server.Write(sslRequest); err != nil {
		log.Printf("ssl request write: %v", err)
		return
	}
	var b [1]byte
	if _, err := io.ReadFull(server, b[:]); err != nil || b[0] != 'S' {
		log.Printf("ssl negotiation: %v (reply %q)", err, string(b[:]))
		return
	}

	tlsCfg, err := tlsConfig()
	if err != nil {
		log.Printf("tls config: %v", err)
		return
	}
	tlsConn := tls.Client(server, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("tls handshake: %v", err)
		return
	}
	if _, err := tlsConn.Write(startup); err != nil {
		log.Printf("startup replay: %v", err)
		return
	}
	log.Printf("upstream connection established (%s)", server.RemoteAddr())

	// Pipe both ways; when either direction ends, tear both down.
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(tlsConn, client); done <- struct{}{} }()
	go func() { _, _ = io.Copy(client, tlsConn); done <- struct{}{} }()
	<-done
}

// tlsConfig loads the SVID (client cert + key) and the server CA bundle
// fresh on every call, so SVID rotation is picked up per connection.
func tlsConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(svidCert, svidKey)
	if err != nil {
		return nil, err
	}
	ca, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, io.ErrUnexpectedEOF
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   "postgres-rw.database.svc",
		MinVersion:   tls.VersionTLS12,
	}, nil
}
