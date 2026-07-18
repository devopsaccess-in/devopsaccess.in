import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import { blog } from "#velite";

export const metadata: Metadata = {
  title: "Blog — devopsaccess",
  description:
    "Deep technical notes on Kubernetes, CI/CD, cloud cost and platform engineering.",
  alternates: { canonical: "/blog/" },
};

const fmt = (d: string) =>
  new Date(d).toLocaleDateString("en-GB", { year: "numeric", month: "short", day: "numeric" });

export default function BlogIndex() {
  const posts = [...blog].sort(
    (a, b) => new Date(b.pubDate).valueOf() - new Date(a.pubDate).valueOf(),
  );

  return (
    <>
      <SiteJsonLd pathname="/blog/" />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Blog</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Field notes from the control plane.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          Long-form, opinionated writing about the work — what broke, what fixed it, and what
          we&apos;d do differently.
        </p>
      </section>

      <section className="container-px pb-16">
        <ul className="divide-y divide-ink-line border-y border-ink-line">
          {posts.map((post) => (
            <li key={post.slug}>
              <Link href={`/blog/${post.slug}/`} className="group block py-6">
                <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
                  <time className="font-mono text-xs text-mist-dim">{fmt(post.pubDate)}</time>
                  {post.readingTime && (
                    <span className="font-mono text-xs text-mist-dim">· {post.readingTime}</span>
                  )}
                </div>
                <h2 className="mt-1 text-xl text-white transition-colors group-hover:text-node">
                  {post.title}
                </h2>
                <p className="prose-body mt-1 max-w-prose text-sm">{post.description}</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {post.tags.map((t) => (
                    <span
                      key={t}
                      className="rounded border border-ink-line px-2 py-0.5 font-mono text-[11px] text-mist-dim"
                    >
                      {t}
                    </span>
                  ))}
                </div>
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
