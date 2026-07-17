import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "DevOps Access", template: "%s — DevOps Access" },
  description: "Uptime monitoring and alerting dashboard",
  robots: { index: false },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
