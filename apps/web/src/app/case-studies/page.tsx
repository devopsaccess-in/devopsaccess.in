import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import { caseStudies } from "#velite";

export const metadata: Metadata = {
  title: "Case Studies — devopsaccess",
  description: "Anonymized accounts of real DevOps engagements: the problem, the work, the result.",
  alternates: { canonical: "/case-studies/" },
};

export default function CaseStudiesIndex() {
  const studies = [...caseStudies].sort(
    (a, b) => new Date(b.pubDate).valueOf() - new Date(a.pubDate).valueOf(),
  );

  return (
    <>
      <SiteJsonLd pathname="/case-studies/" />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Case Studies</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          The work, with the names filed off.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          Real engagements, anonymized to respect client confidentiality. Same problems
          you&apos;re probably facing.
        </p>
      </section>

      <section className="container-px pb-16">
        <div className="grid gap-5 sm:grid-cols-2">
          {studies.map((s) => (
            <Link key={s.slug} href={`/case-studies/${s.slug}/`} className="card group block">
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs text-node">{s.sector}</span>
                <span className="status-pill !px-2 !py-0.5">
                  <span className="status-dot" />
                  resolved
                </span>
              </div>
              <h2 className="mt-3 text-xl group-hover:text-node">{s.client}</h2>
              <p className="prose-body mt-2 text-sm">{s.challenge}</p>
              <div className="mt-4 flex flex-wrap gap-2">
                {s.stack.map((t) => (
                  <span
                    key={t}
                    className="rounded border border-ink-line px-2 py-0.5 font-mono text-[11px] text-mist-dim"
                  >
                    {t}
                  </span>
                ))}
              </div>
              <p className="mt-4 font-mono text-xs text-signal">→ {s.result}</p>
            </Link>
          ))}
        </div>
      </section>
    </>
  );
}
