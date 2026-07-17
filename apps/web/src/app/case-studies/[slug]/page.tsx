import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import SiteJsonLd from "@/components/SiteJsonLd";
import { caseStudyArticle } from "@/lib/schema";
import { abs } from "@/lib/site";
import { caseStudies } from "#velite";

type Params = Promise<{ slug: string }>;

export function generateStaticParams() {
  return caseStudies.map((study) => ({ slug: study.slug }));
}

export async function generateMetadata({ params }: { params: Params }): Promise<Metadata> {
  const { slug } = await params;
  const study = caseStudies.find((s) => s.slug === slug);
  if (!study) return {};
  return {
    title: `${study.client} — Case Study — devopsaccess`,
    description: study.challenge,
    alternates: { canonical: `/case-studies/${study.slug}/` },
  };
}

export default async function CaseStudy({ params }: { params: Params }) {
  const { slug } = await params;
  const study = caseStudies.find((s) => s.slug === slug);
  if (!study) notFound();

  const pathname = `/case-studies/${study.slug}/`;
  const articleSchema = caseStudyArticle(abs(pathname), {
    client: study.client,
    challenge: study.challenge,
    result: study.result,
    pubDate: new Date(study.pubDate),
  });

  return (
    <>
      <SiteJsonLd pathname={pathname} extra={[articleSchema]} />

      <article className="container-px py-16">
        <Link href="/case-studies" className="font-mono text-xs text-mist-dim hover:text-node">
          ← all case studies
        </Link>

        <header className="mt-6 max-w-prose">
          <span className="font-mono text-xs text-node">{study.sector}</span>
          <h1 className="mt-2 text-3xl font-bold sm:text-4xl">{study.client}</h1>
        </header>

        <div className="mt-8 grid gap-4 sm:grid-cols-3">
          <div className="rounded-lg border border-ink-line bg-ink-card/40 p-4">
            <div className="font-mono text-xs uppercase tracking-wide text-mist-dim">
              Challenge
            </div>
            <p className="mt-1 text-sm text-mist">{study.challenge}</p>
          </div>
          <div className="rounded-lg border border-ink-line bg-ink-card/40 p-4">
            <div className="font-mono text-xs uppercase tracking-wide text-mist-dim">Stack</div>
            <div className="mt-1 flex flex-wrap gap-1.5">
              {study.stack.map((t) => (
                <span
                  key={t}
                  className="rounded border border-ink-line px-1.5 py-0.5 font-mono text-[11px] text-mist-dim"
                >
                  {t}
                </span>
              ))}
            </div>
          </div>
          <div className="rounded-lg border border-node/30 bg-node/[0.04] p-4">
            <div className="font-mono text-xs uppercase tracking-wide text-node">Result</div>
            <p className="mt-1 text-sm text-mist">{study.result}</p>
          </div>
        </div>

        <div
          className="post-body mt-10 max-w-prose"
          dangerouslySetInnerHTML={{ __html: study.html }}
        />

        <div className="mt-12 rounded-xl border border-node/30 bg-node/[0.04] p-6 text-center">
          <p className="prose-body">Facing something similar?</p>
          <Link href="/contact" className="btn-primary mt-3">
            Book a discovery call
          </Link>
        </div>
      </article>
    </>
  );
}
