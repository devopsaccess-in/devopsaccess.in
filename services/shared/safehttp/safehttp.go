// Package safehttp provides SSRF-guarded HTTP dialing for anything that
// connects to customer-supplied URLs (the uptime prober, URL validation in the
// API). It never connects to private/loopback/link-local addresses — otherwise
// a customer could point a monitor at our own localhost services (Postgres,
// Grafana, the API itself, cloud metadata). The *resolved* IP is validated at
// connect time via Dialer.Control, which also defeats DNS rebinding; only
// ports 80/443 are allowed.
package safehttp

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// IsPublicIP reports whether ip is a publicly routable address.
func IsPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	return true
}

// Dialer validates every connection's resolved IP + port before connecting.
func Dialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout: timeout,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, port, err := net.SplitHostPort(address) // address is the resolved IP:port
			if err != nil {
				return err
			}
			if port != "80" && port != "443" {
				return fmt.Errorf("port %s not allowed", port)
			}
			if !IsPublicIP(net.ParseIP(host)) {
				return fmt.Errorf("blocked non-public address %s", host)
			}
			return nil
		},
	}
}

// Client returns an HTTP client that fetches public URLs only, with bounded
// redirects and an overall timeout.
func Client(timeout time.Duration) *http.Client {
	d := Dialer(5 * time.Second)
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           d.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: timeout,
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
