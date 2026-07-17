import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import SiteJsonLd from "@/components/SiteJsonLd";
import { blogPosting } from "@/lib/schema";
import { abs } from "@/lib/site";
import { blog } from "#velite";

type Params = Promise<{ slug: string }>;

export function generateStaticParams() {
  return blog.map((post) => ({ slug: post.slug }));
}

export async function generateMetadata({ params }: { params: Params }): Promise<Metadata> {
  const { slug } = await params;
  const post = blog.find((p) => p.slug === slug);
  if (!post) return {};
  return {
    title: `${post.title} — devopsaccess`,
    description: post.description,
    alternates: { canonical: `/blog/${post.slug}/` },
  };
}

const fmt = (d: string) =>
  new Date(d).toLocaleDateString("en-GB", { year: "numeric", month: "long", day: "numeric" });

export default async function BlogPost({ params }: { params: Params }) {
  const { slug } = await params;
  const post = blog.find((p) => p.slug === slug);
  if (!post) notFound();

  const pathname = `/blog/${post.slug}/`;
  const articleSchema = blogPosting(abs(pathname), {
    title: post.title,
    description: post.description,
    pubDate: new Date(post.pubDate),
    updated: post.updated ? new Date(post.updated) : undefined,
  });

  return (
    <>
      <SiteJsonLd pathname={pathname} extra={[articleSchema]} />

      <article className="container-px py-16">
        <Link href="/blog" className="font-mono text-xs text-mist-dim hover:text-node">
          ← all posts
        </Link>
        <header className="mt-6 max-w-prose">
          <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
            <time className="font-mono text-xs text-node">{fmt(post.pubDate)}</time>
            {post.readingTime && (
              <span className="font-mono text-xs text-mist-dim">· {post.readingTime}</span>
            )}
          </div>
          <h1 className="mt-2 text-3xl font-bold sm:text-4xl">{post.title}</h1>
          <p className="prose-body mt-3 text-lg">{post.description}</p>
          <div className="mt-4 flex flex-wrap gap-2">
            {post.tags.map((t) => (
              <span
                key={t}
                className="rounded border border-ink-line px-2 py-0.5 font-mono text-[11px] text-mist-dim"
              >
                {t}
              </span>
            ))}
          </div>
        </header>

        <div
          className="post-body mt-10 max-w-prose"
          dangerouslySetInnerHTML={{ __html: post.html }}
        />
      </article>
    </>
  );
}
