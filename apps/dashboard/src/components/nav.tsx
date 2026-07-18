"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import clsx from "clsx";
import { useMe } from "@/app/(app)/providers";

const links = [
  { href: "/monitors", label: "Monitors" },
  { href: "/incidents", label: "Incidents" },
  { href: "/channels", label: "Channels" },
  { href: "/settings", label: "Settings" },
];

export function Nav() {
  const pathname = usePathname();
  const me = useMe();

  return (
    <header className="border-b border-ink-line bg-ink-soft/60">
      <div className="container-px flex h-14 items-center gap-6">
        <Link href="/monitors" className="font-display font-semibold text-white">
          DevOps<span className="text-node">Access</span>
        </Link>
        <nav className="flex gap-1">
          {links.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className={clsx(
                "rounded-md px-3 py-1.5 text-sm",
                pathname.startsWith(l.href)
                  ? "bg-ink-card text-white"
                  : "text-mist-dim hover:text-mist",
              )}
            >
              {l.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-4">
          {me.data && (
            <span className="hidden font-mono text-xs text-mist-faint sm:inline">
              {me.data.tenant.slug}
            </span>
          )}
          <a href="/auth/logout" className="text-sm text-mist-dim hover:text-mist">
            Sign out
          </a>
        </div>
      </div>
    </header>
  );
}
