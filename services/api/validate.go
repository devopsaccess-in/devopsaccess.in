package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/safehttp"
)

// lookupFunc resolves a hostname; injectable so validation is unit-testable
// without the network.
type lookupFunc func(ctx context.Context, host string) ([]net.IP, error)

func defaultLookup(ctx context.Context, host string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

// validateMonitorURL rejects anything the prober would refuse to fetch, so a
// bad target fails loudly at create time instead of silently at probe time:
// non-http(s) schemes, embedded credentials, non-standard ports, and hosts
// that are (or resolve to) non-public IPs. The prober's dialer re-validates
// every resolved IP at connect time (DNS rebinding defense) — this is the
// early, friendly layer of the same rule. allowPrivate skips the port and
// public-IP checks (E2E-test hook, mirrored in the scheduler).
func validateMonitorURL(ctx context.Context, raw string, lookup lookupFunc, allowPrivate bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("url is not valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must start with http:// or https://")
	}
	if u.User != nil {
		return fmt.Errorf("url must not contain credentials")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}
	if allowPrivate {
		return nil
	}
	if port := u.Port(); port != "" && port != "80" && port != "443" {
		return fmt.Errorf("only ports 80 and 443 are allowed")
	}

	if ip := net.ParseIP(host); ip != nil {
		if !safehttp.IsPublicIP(ip) {
			return fmt.Errorf("host resolves to a non-public address")
		}
		return nil
	}
	ips, err := lookup(ctx, host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve host %q", host)
	}
	for _, ip := range ips {
		if !safehttp.IsPublicIP(ip) {
			return fmt.Errorf("host resolves to a non-public address")
		}
	}
	return nil
}
