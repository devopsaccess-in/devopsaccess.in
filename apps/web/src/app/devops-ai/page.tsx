import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import { tools } from "@/data/tools";
import { itemList } from "@/lib/schema";

export const metadata: Metadata = {
  title: "DevOps & AI tools we rate — devopsaccess",
  description:
    "An opinionated, regularly-updated directory of the DevOps, automation and AI-agent tools we actually recommend — most of them open-source and self-hostable.",
  alternates: { canonical: "/devops-ai/" },
};

const categories = [...new Set(tools.map((t) => t.category))];
const schema = itemList(
  "DevOps & AI tools we rate",
  tools.map((t) => ({ name: t.name, url: t.url, description: t.take })),
);

export default function DevopsAi() {
  return (
    <>
      <SiteJsonLd pathname="/devops-ai/" extra={[schema]} />

      <section className="container-px pt-16 pb-8">
        <p className="eyebrow">DevOps · AI</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">Tools we actually rate.</h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          An opinionated, evolving list of DevOps, automation and AI-agent tools we&apos;d
          recommend to startups — biased toward open-source and self-hostable so you&apos;re never
          locked in or surprised by a per-host bill. Want help wiring any of these into your stack?{" "}
          <Link href="/contact" className="text-node hover:underline">
            Book a call
          </Link>
          .
        </p>
      </section>

      {categories.map((cat) => (
        <section key={cat} className="container-px pb-8">
          <h2 className="text-2xl">{cat}</h2>
          <div className="mt-5 grid gap-5 sm:grid-cols-2">
            {tools
              .filter((t) => t.category === cat)
              .map((t) => (
                <a
                  key={t.name}
                  href={t.url}
                  rel="noopener"
                  target="_blank"
                  className="card group block"
                >
                  <div className="flex items-center justify-between">
                    <h3 className="text-xl">{t.name}</h3>
                    <div className="flex gap-1.5 font-mono text-[11px] text-mist-dim">
                      {t.selfHost && (
                        <span className="rounded border border-ink-line px-1.5 py-0.5">
                          self-host
                        </span>
                      )}
                      {t.free && (
                        <span className="rounded border border-node/40 px-1.5 py-0.5 text-node">
                          free
                        </span>
                      )}
                    </div>
                  </div>
                  <p className="prose-body mt-2 text-sm">{t.take}</p>
                  <span className="mt-3 inline-block font-mono text-xs text-mist-dim group-hover:text-node">
                    visit →
                  </span>
                </a>
              ))}
          </div>
        </section>
      ))}
    </>
  );
}
