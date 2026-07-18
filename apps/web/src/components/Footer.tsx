"use client";

import Link from "next/link";

export default function Footer() {
  const year = new Date().getFullYear();
  return (
    <footer className="mt-24 border-t border-ink-line">
      <div className="container-px flex flex-col gap-6 py-10 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="font-display text-white">
            <span className="font-mono text-node">$</span>devopsaccess
          </div>
          <p className="mt-1 font-mono text-xs text-mist-dim">
            Uptime monitoring &amp; alerting for small teams · {year}
          </p>
        </div>
        <div className="flex flex-wrap gap-x-6 gap-y-2 font-mono text-xs text-mist-dim">
          <Link href="/services" className="hover:text-node">services</Link>
          <Link href="/case-studies" className="hover:text-node">case-studies</Link>
          <Link href="/site-check" className="hover:text-node">free check</Link>
          <Link href="/devops-ai" className="hover:text-node">tools</Link>
          <Link href="/blog" className="hover:text-node">blog</Link>
          <Link href="/about" className="hover:text-node">about</Link>
          <Link href="/contact" className="hover:text-node">contact</Link>
          <Link href="/uses" className="hover:text-node">uses</Link>
        </div>
      </div>
      <div className="container-px flex flex-wrap gap-x-5 gap-y-2 border-t border-ink-line py-5 font-mono text-[0.7rem] text-mist-dim">
        <Link href="/privacy" className="hover:text-node">privacy</Link>
        <Link href="/terms" className="hover:text-node">terms</Link>
        <Link href="/cookie-policy" className="hover:text-node">cookie-policy</Link>
        <Link href="/refund" className="hover:text-node">refund</Link>
        <button
          type="button"
          className="hover:text-node"
          onClick={() => window.dispatchEvent(new Event("da-open-prefs"))}
        >
          cookie preferences
        </button>
      </div>
    </footer>
  );
}
