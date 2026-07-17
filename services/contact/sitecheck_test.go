package main

import (
	"net"
	"net/http"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", "10.0.0.5", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", // cloud metadata
		"0.0.0.0", "fc00::1", "fe80::1",
	}
	for _, s := range blocked {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be blocked (non-public)", s)
		}
	}
	allowed := []string{"1.1.1.1", "8.8.8.8", "104.16.0.1", "2606:4700::1"}
	for _, s := range allowed {
		if !isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be allowed (public)", s)
		}
	}
	if isPublicIP(nil) {
		t.Error("nil IP must be blocked")
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in       string
		wantHost string
		ok       bool
	}{
		{"example.com", "example.com", true},
		{"https://example.com/path", "example.com", true},
		{"http://sub.example.co.uk", "sub.example.co.uk", true},
		{"ftp://example.com", "", false},
		{"", "", false},
		{"   ", "", false},
	}
	for _, c := range cases {
		_, host, err := normalizeURL(c.in)
		if c.ok && (err != nil || host != c.wantHost) {
			t.Errorf("normalizeURL(%q) = host %q err %v, want host %q ok", c.in, host, err, c.wantHost)
		}
		if !c.ok && err == nil {
			t.Errorf("normalizeURL(%q) should have errored", c.in)
		}
	}
}

func TestGrade(t *testing.T) {
	if g := grade(6, 6, nil).Grade; g != "A" {
		t.Errorf("6/6 = %s, want A", g)
	}
	if g := grade(0, 6, nil).Grade; g != "F" {
		t.Errorf("0/6 = %s, want F", g)
	}
	if g := grade(3, 6, nil).Grade; g != "D" {
		t.Errorf("3/6 (50%%) = %s, want D", g)
	}
	if g := grade(5, 6, nil).Grade; g != "B" {
		t.Errorf("5/6 (83%%) = %s, want B", g)
	}
}

func TestCheckSecurity(t *testing.T) {
	h := http.Header{}
	h.Set("Strict-Transport-Security", "max-age=31536000")
	h.Set("X-Content-Type-Options", "nosniff")
	s := checkSecurity(h)
	if s.Score != 2 || s.Max != 6 {
		t.Errorf("security score = %d/%d, want 2/6", s.Score, s.Max)
	}
}
