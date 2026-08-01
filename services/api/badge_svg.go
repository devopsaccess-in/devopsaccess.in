package main

import (
	"fmt"
	"net/http"
	"strings"
)

// writeBadge renders a shields.io-style flat SVG badge and writes it with an
// svg content type and a short cache window (GitHub's camo proxy caches
// aggressively regardless, so the badge is "live-ish", not real-time).
func writeBadge(w http.ResponseWriter, label, value, color string) {
	svg := renderBadge(label, value, color)
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

// renderBadge builds the SVG. Text is XML-escaped and sized with textLength so
// a value never overflows its segment, and the label (which may be
// caller-supplied) can never break out of the markup.
func renderBadge(label, value, color string) string {
	label = xmlEscape(label)
	value = xmlEscape(value)

	const pad = 6 // horizontal padding each side of each segment
	labelTextW := textWidth(label)
	valueTextW := textWidth(value)
	labelW := labelTextW + 2*pad
	valueW := valueTextW + 2*pad
	totalW := labelW + valueW

	// Segment text centers, in 10x units for the scale(.1) crispness trick.
	labelCX := labelW * 5           // (labelW/2) * 10
	valueCX := (labelW + valueW/2) * 10
	labelTL := labelTextW * 10
	valueTL := valueTextW * 10

	aria := fmt.Sprintf("%s: %s", label, value)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s">`+
		`<title>%s</title>`+
		`<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`+
		`<clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>`+
		`<g clip-path="url(#r)">`+
		`<rect width="%d" height="20" fill="%s"/>`+
		`<rect x="%d" width="%d" height="20" fill="%s"/>`+
		`<rect width="%d" height="20" fill="url(#s)"/>`+
		`</g>`+
		`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">`+
		`<text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`+
		`<text x="%d" y="140" transform="scale(.1)" textLength="%d">%s</text>`+
		`<text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="%d">%s</text>`+
		`<text x="%d" y="140" transform="scale(.1)" textLength="%d">%s</text>`+
		`</g></svg>`,
		totalW, aria,
		aria,
		totalW,
		labelW, colorLabel,
		labelW, valueW, color,
		totalW,
		labelCX, labelTL, label,
		labelCX, labelTL, label,
		valueCX, valueTL, value,
		valueCX, valueTL, value,
	)
}

// textWidth approximates the pixel width of s at font-size 11 (Verdana-ish).
// Exactness doesn't matter — textLength forces the glyphs to fit — but a close
// estimate keeps the padding visually even.
func textWidth(s string) int {
	w := 0.0
	for _, r := range s {
		switch {
		case r == 'i' || r == 'l' || r == 'j' || r == '.' || r == ':' || r == '\'' || r == '|':
			w += 3.0
		case r == 'f' || r == 't' || r == 'r' || r == ' ':
			w += 4.0
		case r == 'm' || r == 'w' || r == 'M' || r == 'W' || r == '%':
			w += 9.5
		case r >= 'A' && r <= 'Z':
			w += 7.5
		default:
			w += 6.5
		}
	}
	return int(w + 0.5)
}

func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}
