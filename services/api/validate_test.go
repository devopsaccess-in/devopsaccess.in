package main

import (
	"context"
	"fmt"
	"net"
	"testing"
)

// fakeLookup returns fixed IPs for any hostname; errHost forces a resolution
// failure.
func fakeLookup(ips []string, errHost string) lookupFunc {
	return func(_ context.Context, host string) ([]net.IP, error) {
		if host == errHost {
			return nil, fmt.Errorf("no such host")
		}
		out := make([]net.IP, 0, len(ips))
		for _, s := range ips {
			out = append(out, net.ParseIP(s))
		}
		return out, nil
	}
}

func TestValidateMonitorURL(t *testing.T) {
	public := fakeLookup([]string{"93.184.216.34"}, "unresolvable.example")
	private := fakeLookup([]string{"10.0.0.5"}, "")
	mixed := fakeLookup([]string{"93.184.216.34", "127.0.0.1"}, "")

	tests := []struct {
		name         string
		url          string
		lookup       lookupFunc
		allowPrivate bool
		wantErr      bool
	}{
		{"https public host", "https://example.com/health", public, false, false},
		{"http public host", "http://example.com", public, false, false},
		{"explicit port 443", "https://example.com:443/x", public, false, false},
		{"public IP literal", "https://93.184.216.34/", public, false, false},
		{"ftp scheme", "ftp://example.com", public, false, true},
		{"no scheme", "example.com", public, false, true},
		{"empty", "", public, false, true},
		{"credentials in url", "https://user:pass@example.com", public, false, true},
		{"non-standard port", "https://example.com:8443", public, false, true},
		{"loopback IP literal", "http://127.0.0.1/", public, false, true},
		{"private IP literal", "http://10.1.2.3/", public, false, true},
		{"link-local IP literal", "http://169.254.169.254/latest/meta-data", public, false, true},
		{"ipv6 loopback", "http://[::1]/", public, false, true},
		{"host resolves private", "https://internal.example.com", private, false, true},
		{"host resolves mixed public+private", "https://rebind.example.com", mixed, false, true},
		{"unresolvable host", "https://unresolvable.example", public, false, true},

		// E2E-test hook: allowPrivate skips port + public-IP checks but keeps
		// scheme/credential rules.
		{"allowPrivate loopback with port", "http://127.0.0.1:39321/x", public, true, false},
		{"allowPrivate still rejects ftp", "ftp://127.0.0.1/", public, true, true},
		{"allowPrivate still rejects credentials", "https://u:p@example.com", public, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMonitorURL(context.Background(), tt.url, tt.lookup, tt.allowPrivate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateMonitorURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
