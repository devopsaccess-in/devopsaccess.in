import localFont from "next/font/local";

// Self-hosted variable fonts (vendored woff2, latin subset — same files
// @fontsource-variable ships). next/font generates metric-matched fallback
// @font-faces automatically (size-adjust/ascent-override), so the swap from
// the system fallback causes zero layout shift — the job Fontaine did in the
// Astro build.

export const spaceGrotesk = localFont({
  src: "../fonts/space-grotesk-latin-wght-normal.woff2",
  weight: "300 700",
  display: "swap",
  variable: "--font-display",
});

export const inter = localFont({
  src: "../fonts/inter-latin-wght-normal.woff2",
  weight: "100 900",
  display: "swap",
  variable: "--font-body",
});

export const jetbrainsMono = localFont({
  src: "../fonts/jetbrains-mono-latin-wght-normal.woff2",
  weight: "100 800",
  display: "swap",
  variable: "--font-mono",
  // Arial metrics make a poor stand-in for a monospace face; the mono font is
  // only used for small labels, so fallback shift is negligible.
  adjustFontFallback: false,
});
