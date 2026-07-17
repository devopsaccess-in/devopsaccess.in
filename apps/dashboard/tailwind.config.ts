import type { Config } from "tailwindcss";

export default {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // Same "control plane" identity as apps/web (keep in sync).
        ink: {
          DEFAULT: "#0a0e1a",
          soft: "#10172a",
          card: "#141d33",
          line: "#1f2b47",
        },
        signal: {
          DEFAULT: "#f5a623", // amber — degraded/warning + CTA
          dim: "#b9791a",
        },
        node: {
          DEFAULT: "#2dd4bf", // teal — healthy/up, links
          dim: "#14b8a6",
        },
        mist: {
          DEFAULT: "#c7d2e3",
          dim: "#8493ad",
          faint: "#5a6781",
        },
        // Status-only red (down/incident). Never used as a series color.
        alert: {
          DEFAULT: "#fb7185",
          dim: "#e11d48",
        },
      },
      fontFamily: {
        display: ["system-ui", "sans-serif"],
        body: ["system-ui", "sans-serif"],
        mono: ["ui-monospace", "SFMono-Regular", "monospace"],
      },
    },
  },
  plugins: [],
} satisfies Config;
