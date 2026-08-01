package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// diagnose turns a transport error into (phase, human-readable cause). The
// message NEVER contains the target URL: causes are stored on incidents and
// shown in alerts, and a URL can carry secret tokens or internal hostnames.
// dnsDone/tcpDone/tlsDone say which phases completed, which is what lets us
// name the phase that broke even when the error itself is vague.
func diagnose(err error, rawURL string, dnsDone, tcpDone, tlsDone bool, timeout time.Duration) (phase, message string) {
	// --- TLS certificate problems (the most valuable diagnosis) ---
	var certInvalid x509.CertificateInvalidError
	if errors.As(err, &certInvalid) {
		switch certInvalid.Reason {
		case x509.Expired:
			if certInvalid.Cert != nil {
				return "tls", "TLS certificate expired " + humanAgo(certInvalid.Cert.NotAfter)
			}
			return "tls", "TLS certificate has expired"
		case x509.NotAuthorizedToSign, x509.CANotAuthorizedForThisName:
			return "tls", "TLS certificate chain is invalid"
		default:
			return "tls", "TLS certificate is not valid"
		}
	}
	var hostErr x509.HostnameError
	if errors.As(err, &hostErr) {
		return "tls", "TLS certificate does not match the requested hostname"
	}
	var authErr x509.UnknownAuthorityError
	if errors.As(err, &authErr) {
		return "tls", "TLS certificate is not trusted (unknown certificate authority)"
	}

	// --- SSRF guard rejections (our own dialer) ---
	msg := err.Error()
	if strings.Contains(msg, "blocked non-public address") {
		return "blocked", "target resolves to a non-public address and was not contacted"
	}
	if strings.Contains(msg, "not allowed") && strings.Contains(msg, "port") {
		return "blocked", "target port is not allowed (only 80 and 443)"
	}
	if strings.Contains(msg, "too many redirects") {
		return "redirect", "too many redirects"
	}

	// --- DNS ---
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return "dns", "DNS lookup failed: host not found"
		}
		if dnsErr.IsTimeout {
			return "dns", "DNS lookup timed out"
		}
		return "dns", "DNS lookup failed"
	}

	// --- Connection refused / reset ---
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "tcp", "connection refused"
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return "tcp", "connection reset by peer"
	}
	if errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) {
		return "tcp", "host unreachable"
	}

	// --- Timeouts: name the phase that was still pending ---
	var netErr net.Error
	isTimeout := errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
	if isTimeout {
		switch {
		case !dnsDone:
			return "dns", "timed out during DNS lookup after " + humanDur(timeout)
		case !tcpDone:
			return "tcp", "timed out connecting after " + humanDur(timeout)
		case isHTTPS(rawURL) && !tlsDone:
			return "tls", "timed out during the TLS handshake after " + humanDur(timeout)
		default:
			return "timeout", "timed out waiting for a response after " + humanDur(timeout)
		}
	}

	// --- Fall back to the furthest phase reached ---
	switch {
	case !dnsDone:
		return "dns", "DNS lookup failed"
	case !tcpDone:
		return "tcp", "could not establish a TCP connection"
	case isHTTPS(rawURL) && !tlsDone:
		return "tls", "TLS handshake failed"
	default:
		return "response", "no valid HTTP response"
	}
}

func isHTTPS(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Scheme == "https"
}

// humanAgo renders how long ago t was, e.g. "3 days ago", "5 hours ago".
func humanAgo(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "in " + humanDur(-d)
	}
	return humanDur(d) + " ago"
}

// humanDur renders a coarse, readable duration ("2 days", "3 hours", "10s").
func humanDur(d time.Duration) string {
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	case d >= 2*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	case d >= 2*time.Minute:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	case d >= time.Second:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
}
