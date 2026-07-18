package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// SSRF guard. The site-check endpoint fetches a user-supplied URL, so it must
// never connect to private/loopback/link-local addresses — otherwise it could be
// pointed at our own localhost services (Postgres, Grafana, Prometheus, the
// contact service, cloud metadata, etc.). We validate the *resolved* IP at
// connect time (via Dialer.Control), which also defeats DNS rebinding, and only
// allow ports 80/443.

func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	return true
}

// safeDialer validates every connection's resolved IP + port before connecting.
func safeDialer() *net.Dialer {
	return &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, port, err := net.SplitHostPort(address) // address is the resolved IP:port
			if err != nil {
				return err
			}
			if port != "80" && port != "443" {
				return fmt.Errorf("port %s not allowed", port)
			}
			if !isPublicIP(net.ParseIP(host)) {
				return fmt.Errorf("blocked non-public address %s", host)
			}
			return nil
		},
	}
}

// safeHTTPClient fetches public URLs only, with bounded redirects + timeout.
func safeHTTPClient() *http.Client {
	d := safeDialer()
	return &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			DialContext:           d.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 8 * time.Second,
			DisableKeepAlives:     true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil // each hop is re-validated by the dialer's Control func
		},
	}
}

// safeTLSDial opens a validated TLS connection to host:443 for cert inspection.
func safeTLSDial(ctx context.Context, host string) (*tls.ConnectionState, error) {
	dialer := safeDialer()
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"),
		&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	st := conn.ConnectionState()
	return &st, nil
}
