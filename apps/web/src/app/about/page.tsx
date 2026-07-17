import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";

export const metadata: Metadata = {
  title: "About — devopsaccess",
  description: "The story, certifications and the GKE/GCP experience behind devopsaccess.",
  alternates: { canonical: "/about/" },
};

const certs = ["Certified Kubernetes Administrator (CKA)"];

const timeline = [
  {
    when: "Now",
    what: "Building DevOps Access in the open — affordable uptime monitoring and alerting for small teams, plus a boutique practice helping startups put their infrastructure under version control.",
  },
  {
    when: "Platform engineering",
    what: "Built internal developer platforms on GKE: self-service namespaces, golden CI templates, and cost guardrails that engineers actually liked.",
  },
  {
    when: "SRE & cloud",
    what: "Years operating production on GCP and AWS — incident command, on-call hygiene, and the slow craft of making systems boring.",
  },
];

export default function About() {
  return (
    <>
      <SiteJsonLd pathname="/about/" />

      <section className="container-px pt-16 pb-12">
        <p className="eyebrow">About</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          I make infrastructure boring — on purpose.
        </h1>
        <p className="prose-body mt-6 max-w-prose text-lg">
          The best compliment infrastructure can get is silence. No 3am pages, no mystery bills, no
          &ldquo;only one person knows how this works.&rdquo; devopsaccess exists to get teams
          there: production that&apos;s reproducible, observable, and handed over so completely you
          could fire me and be fine.
        </p>
        <p className="prose-body mt-4 max-w-prose">
          My background is hands-on GKE and GCP work — clusters, pipelines, migrations and the cost
          cleanup that follows. This very site is part of the pitch: provisioned, configured and
          deployed entirely as code. If you want to see how I work before we talk, read{" "}
          <Link href="/uses" className="text-node hover:underline">
            /uses
          </Link>{" "}
          or the{" "}
          <Link href="/blog" className="text-node hover:underline">
            blog
          </Link>
          .
        </p>
        <p className="prose-body mt-4 max-w-prose">
          Why devopsaccess? Most startups can&apos;t justify a full DevOps hire, so the work falls
          on whoever&apos;s least busy — and it shows up as wasted spend and late-night outages.
          I&apos;m building devopsaccess in the open to fix that: free tools first, then affordable
          uptime monitoring and alerting so small teams get senior-grade operations without hiring
          for it.{" "}
          <Link href="/waitlist" className="text-node hover:underline">
            Join early access
          </Link>{" "}
          to follow along.
        </p>
      </section>

      <section className="container-px pb-12">
        <h2 className="text-2xl">How I got here</h2>
        <ol className="mt-6 space-y-5 border-l border-ink-line pl-6">
          {timeline.map((t) => (
            <li key={t.when} className="relative">
              <span className="absolute -left-[1.65rem] top-1.5 h-3 w-3 rounded-full border-2 border-node bg-ink" />
              <div className="font-mono text-xs uppercase tracking-wide text-node">{t.when}</div>
              <p className="prose-body mt-1 max-w-prose">{t.what}</p>
            </li>
          ))}
        </ol>
      </section>

      <section className="container-px pb-8">
        <h2 className="text-2xl">Certifications</h2>
        <div className="mt-5 grid gap-3 sm:grid-cols-2">
          {certs.map((c) => (
            <div
              key={c}
              className="flex items-center gap-3 rounded-lg border border-ink-line bg-ink-card/40 px-4 py-3"
            >
              <span className="font-mono text-node">✓</span>
              <span className="text-sm text-mist">{c}</span>
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
