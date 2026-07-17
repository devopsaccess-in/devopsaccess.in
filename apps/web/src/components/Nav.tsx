"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/services", label: "Services" },
  { href: "/case-studies", label: "Case Studies" },
  { href: "/devops-ai", label: "Tools" },
  { href: "/blog", label: "Blog" },
  { href: "/about", label: "About" },
];

export default function Nav() {
  const path = usePathname();
  return (
    <header className="sticky top-0 z-50 border-b border-ink-line bg-ink/80 backdrop-blur">
      <nav className="container-px flex h-16 items-center justify-between">
        <Link href="/" className="flex items-center gap-2 font-display text-lg font-bold text-white">
          <span className="font-mono text-node">$</span>devopsaccess
        </Link>

        <div className="hidden items-center gap-7 md:flex">
          {links.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className={`text-sm transition-colors ${
                path.startsWith(l.href) ? "text-node" : "text-mist-dim hover:text-mist"
              }`}
            >
              {l.label}
            </Link>
          ))}
          <Link href="/waitlist" className="btn-primary !px-4 !py-1.5 text-sm">
            Join early access
          </Link>
        </div>

        <Link href="/waitlist" className="btn-primary !px-4 !py-1.5 text-sm md:hidden">
          Early access
        </Link>
      </nav>
    </header>
  );
}
