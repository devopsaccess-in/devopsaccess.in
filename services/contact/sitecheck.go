package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/net/html"
)

type sitecheckHandler struct {
	store     *store
	turnstile *turnstileVerifier
	limiter   *ipLimiter
	log       zerolog.Logger
}

type sitecheckRequest struct {
	URL       string `json:"url"`
	Owns      bool   `json:"owns"`
	Website   string `json:"website"`   // honeypot
	Turnstile string `json:"turnstile"` // Cloudflare Turnstile token
}

type checkItem struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type section struct {
	Grade string      `json:"grade"`
	Score int         `json:"score"`
	Max   int         `json:"max"`
	Items []checkItem `json:"items"`
}

type sitecheckResponse struct {
	URL       string  `json:"url"`
	Host      string  `json:"host"`
	CheckedAt string  `json:"checkedAt"`
	Security  section `json:"security"`
	TLS       section `json:"tls"`
	SEO       section `json:"seo"`
}

func grade(score, max int, items []checkItem) section {
	pct := 0
	if max > 0 {
		pct = score * 100 / max
	}
	g := "F"
	switch {
	case pct >= 90:
		g = "A"
	case pct >= 75:
		g = "B"
	case pct >= 55:
		g = "C"
	case pct >= 35:
		g = "D"
	}
	return section{Grade: g, Score: score, Max: max, Items: items}
}

func (h *sitecheckHandler) handle(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.limiter.allow(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many checks. Please try again later."})
		return
	}

	var req sitecheckRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request."})
		return
	}
	if strings.TrimSpace(req.Website) != "" { // honeypot
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if !req.Owns {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please confirm you own this site or have permission to scan it."})
		return
	}

	target, host, err := normalizeURL(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter a valid website URL (e.g. example.com)."})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	ok, codes, terr := h.turnstile.verify(ctx, req.Turnstile, ip)
	if terr != nil {
		h.log.Error().Err(terr).Msg("turnstile verify error")
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Verification temporarily unavailable. Please try again."})
		return
	}
	if !ok {
		h.log.Warn().Strs("turnstile_codes", codes).Str("ip", ip).Msg("turnstile rejected (sitecheck)")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Verification failed. Please retry."})
		return
	}

	client := safeHTTPClient()

	// Fetch the page.
	body, headers, err := fetchPage(ctx, client, target)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Couldn't reach that site: " + err.Error()})
		return
	}

	resp := sitecheckResponse{
		URL:       target,
		Host:      host,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Security:  checkSecurity(headers),
		TLS:       checkTLS(ctx, host),
		SEO:       checkSEO(ctx, client, target, host, body),
	}

	// Best-effort: record what people scan (a demand signal), never fail on it.
	if h.store != nil {
		if err := h.store.recordScan(ctx, host, resp.Security.Grade, resp.TLS.Grade, resp.SEO.Grade, ip); err != nil {
			h.log.Error().Err(err).Msg("record scan failed")
		}
	}
	h.log.Info().Str("host", host).Str("sec", resp.Security.Grade).Str("tls", resp.TLS.Grade).Str("seo", resp.SEO.Grade).Msg("site check")

	writeJSON(w, http.StatusOK, resp)
}

// normalizeURL accepts "example.com" or a full URL; returns a safe https(.)/http
// URL and the hostname.
func normalizeURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", fmt.Errorf("scheme not allowed")
	}
	return u.Scheme + "://" + u.Host + u.Path, u.Hostname(), nil
}

func fetchPage(ctx context.Context, c *http.Client, target string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "devopsaccess-sitecheck/1.0 (+https://devopsaccess.in)")
	res, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20)) // 2 MB cap
	return body, res.Header, nil
}

func checkSecurity(h http.Header) section {
	defs := []struct{ name, header string }{
		{"HSTS (Strict-Transport-Security)", "Strict-Transport-Security"},
		{"Content-Security-Policy", "Content-Security-Policy"},
		{"X-Content-Type-Options", "X-Content-Type-Options"},
		{"X-Frame-Options", "X-Frame-Options"},
		{"Referrer-Policy", "Referrer-Policy"},
		{"Permissions-Policy", "Permissions-Policy"},
	}
	items := make([]checkItem, 0, len(defs))
	score := 0
	for _, d := range defs {
		present := h.Get(d.header) != ""
		if present {
			score++
		}
		detail := "missing"
		if present {
			detail = "present"
		}
		items = append(items, checkItem{d.name, present, detail})
	}
	return grade(score, len(defs), items)
}

func checkTLS(ctx context.Context, host string) section {
	st, err := safeTLSDial(ctx, host)
	if err != nil || st == nil || len(st.PeerCertificates) == 0 {
		reason := "no valid HTTPS/TLS"
		if err != nil {
			reason = err.Error()
		}
		return section{Grade: "F", Score: 0, Max: 3, Items: []checkItem{{"HTTPS/TLS", false, reason}}}
	}
	cert := st.PeerCertificates[0]
	items := make([]checkItem, 0, 3)
	score := 0

	modern := st.Version >= tls.VersionTLS12
	if modern {
		score++
	}
	items = append(items, checkItem{"Protocol", modern, tlsVersionName(st.Version)})

	days := int(time.Until(cert.NotAfter).Hours() / 24)
	expOK := days > 14
	if expOK {
		score++
	}
	items = append(items, checkItem{"Certificate expiry", expOK, fmt.Sprintf("%d days remaining", days)})

	// Reaching here means Go verified the chain + hostname (we didn't skip verify).
	score++
	items = append(items, checkItem{"Trusted certificate", true, "valid and matches the hostname"})

	return grade(score, 3, items)
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1 (outdated)"
	case tls.VersionTLS10:
		return "TLS 1.0 (outdated)"
	default:
		return "unknown"
	}
}

func checkSEO(ctx context.Context, c *http.Client, target, host string, body []byte) section {
	p := parseHTML(body)
	items := make([]checkItem, 0, 9)
	score, max := 0, 0

	add := func(name string, ok bool, detail string) {
		max++
		if ok {
			score++
		}
		items = append(items, checkItem{name, ok, detail})
	}

	titleOK := len(p.title) >= 10 && len(p.title) <= 70
	add("Title tag", titleOK, fmt.Sprintf("%d chars (ideal 10–70)", len(p.title)))
	descOK := len(p.description) >= 50 && len(p.description) <= 160
	add("Meta description", descOK, fmt.Sprintf("%d chars (ideal 50–160)", len(p.description)))
	add("Canonical URL", p.canonical, presence(p.canonical))
	add("Open Graph (title + image)", p.ogTitle && p.ogImage, presence(p.ogTitle && p.ogImage))
	add("Viewport (mobile)", p.viewport, presence(p.viewport))
	add("HTML lang attribute", p.lang, presence(p.lang))
	add("One H1 heading", p.h1, presence(p.h1))

	robotsOK, sitemap := checkRobots(ctx, c, target, host, body)
	add("robots.txt", robotsOK, presence(robotsOK))
	add("Sitemap referenced", sitemap, presence(sitemap))

	return grade(score, max, items)
}

func presence(ok bool) string {
	if ok {
		return "present"
	}
	return "missing"
}

func checkRobots(ctx context.Context, c *http.Client, target, host string, body []byte) (bool, bool) {
	scheme := "https"
	if strings.HasPrefix(target, "http://") {
		scheme = "http"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+host+"/robots.txt", nil)
	if err != nil {
		return false, false
	}
	res, err := c.Do(req)
	if err != nil {
		return false, false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false, false
	}
	rb, _ := io.ReadAll(io.LimitReader(res.Body, 256<<10))
	hasSitemap := strings.Contains(strings.ToLower(string(rb)), "sitemap")
	return true, hasSitemap
}

type htmlFacts struct {
	title       string
	description string
	canonical   bool
	ogTitle     bool
	ogImage     bool
	viewport    bool
	lang        bool
	h1          bool
}

func parseHTML(body []byte) htmlFacts {
	var f htmlFacts
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return f
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				if attr(n, "lang") != "" {
					f.lang = true
				}
			case "title":
				if n.FirstChild != nil {
					f.title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "h1":
				f.h1 = true
			case "link":
				if strings.EqualFold(attr(n, "rel"), "canonical") {
					f.canonical = true
				}
			case "meta":
				name := strings.ToLower(attr(n, "name"))
				prop := strings.ToLower(attr(n, "property"))
				switch {
				case name == "description":
					f.description = strings.TrimSpace(attr(n, "content"))
				case name == "viewport":
					f.viewport = true
				case prop == "og:title":
					f.ogTitle = true
				case prop == "og:image":
					f.ogImage = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return f
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
