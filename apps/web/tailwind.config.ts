import type { Config } from "tailwindcss";

export default {
  content: ["./src/**/*.{ts,tsx,mdx}", "./content/**/*.mdx"],
  theme: {
    extend: {
      colors: {
        // Identity: "control plane" — deep slate substrate, signal amber,
        // healthy node teal. Not the acid-green default.
        ink: {
          DEFAULT: "#0a0e1a",
          soft: "#10172a",
          card: "#141d33",
          line: "#1f2b47",
        },
        signal: {
          DEFAULT: "#f5a623", // amber — alerts/CTA, used sparingly
          dim: "#b9791a",
        },
        node: {
          DEFAULT: "#2dd4bf", // teal — "healthy" status, links
          dim: "#14b8a6",
        },
        mist: {
          DEFAULT: "#c7d2e3",
          dim: "#8493ad",
          faint: "#5a6781",
        },
      },
      fontFamily: {
        // Wired to next/font/local CSS variables (src/lib/fonts.ts), which
        // include auto-generated metric-matched fallbacks — zero CLS on swap.
        display: ["var(--font-display)", "system-ui", "sans-serif"],
        body: ["var(--font-body)", "system-ui", "sans-serif"],
        mono: ["var(--font-mono)", "ui-monospace", "monospace"],
      },
      maxWidth: {
        prose: "68ch",
      },
    },
  },
  plugins: [],
} satisfies Config;
