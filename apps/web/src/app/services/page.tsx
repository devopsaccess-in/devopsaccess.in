import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import PaymentButton from "@/components/PaymentButton";
import { serviceList } from "@/lib/schema";

export const metadata: Metadata = {
  title: "Services — devopsaccess",
  description: "Kubernetes, CI/CD, cloud migration and FinOps, delivered as code.",
  alternates: { canonical: "/services/" },
};

const services = [
  {
    code: "k8s",
    title: "Kubernetes",
    summary:
      "Clusters you can reason about. We design, migrate and operate Kubernetes so it stops being the scariest part of your stack.",
    points: [
      "Cluster architecture for GKE, EKS and self-managed",
      "Multi-cluster and multi-region topologies",
      "GitOps delivery with Argo CD or Flux",
      "Autoscaling, resource policy, and noisy-neighbor isolation",
      "Production readiness reviews and runbooks",
    ],
  },
  {
    code: "cicd",
    title: "CI/CD",
    summary:
      "Pipelines that earn trust. Fast feedback, safe rollouts, and a clear story for every artifact that reaches production.",
    points: [
      "Pipeline design for GitHub Actions, GitLab CI, Argo Workflows",
      "Progressive delivery: canary, blue/green, feature flags",
      "Build caching and test parallelization",
      "Supply-chain security: SBOMs, signing, provenance",
      "One-command rollback you can run under pressure",
    ],
  },
  {
    code: "migrate",
    title: "Cloud Migration",
    summary:
      "From snowflake servers to reproducible infrastructure — without a scary big-bang weekend.",
    points: [
      "Discovery and dependency mapping",
      "Terraform-first landing zones on GCP / AWS",
      "Incremental, zero-downtime cutover plans",
      "Data migration strategy and validation",
      "Documentation and team enablement on handover",
    ],
  },
  {
    code: "finops",
    title: "FinOps",
    summary:
      "Stop paying for idle. We find the waste, right-size what's left, and install guardrails so cost stays honest.",
    points: [
      "Cost visibility by team, service and environment",
      "Right-sizing requests/limits and node pools",
      "Committed-use and spot/preemptible strategy",
      "Idle-resource detection and cleanup automation",
      "Budgets, alerts and showback that engineers respect",
    ],
  },
];

export default function Services() {
  return (
    <>
      <SiteJsonLd pathname="/services/" extra={[serviceList(services)]} />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Services</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Engagements scoped to outcomes, not hours.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          Every engagement ends with you owning reproducible infrastructure and the knowledge to
          run it. No lock-in to us — that&apos;s the point.
        </p>
      </section>

      <section className="container-px space-y-5 pb-16">
        {services.map((s) => (
          <article key={s.code} className="card">
            <div className="flex items-baseline gap-3">
              <span className="font-mono text-xs text-node">{s.code}</span>
              <h2 className="text-2xl">{s.title}</h2>
            </div>
            <p className="prose-body mt-2 max-w-prose">{s.summary}</p>
            <ul className="mt-5 grid gap-2 sm:grid-cols-2">
              {s.points.map((p) => (
                <li key={p} className="flex gap-2 text-sm text-mist">
                  <span className="mt-1 text-node">▸</span>
                  <span>{p}</span>
                </li>
              ))}
            </ul>
          </article>
        ))}
      </section>

      {/* Paid strategy session (Razorpay) */}
      <section className="container-px pb-8">
        <div className="rounded-xl border border-ink-line bg-ink-card/50 p-8 sm:flex sm:items-center sm:justify-between">
          <div className="max-w-prose">
            <p className="eyebrow">Paid deep-dive</p>
            <h2 className="mt-2 text-2xl">Book a paid strategy session</h2>
            <p className="prose-body mt-3">
              A focused, paid 90-minute working session: we go deep on one problem — architecture,
              a migration plan, or a cost teardown — and you leave with concrete next steps. Fee is
              credited if we go on to a full engagement.
            </p>
          </div>
          <div className="mt-6 shrink-0 sm:mt-0 sm:pl-8">
            <PaymentButton />
          </div>
        </div>
      </section>

      <section className="container-px pb-8">
        <div className="rounded-xl border border-node/30 bg-node/[0.04] p-8 text-center">
          <h2 className="text-2xl">Not sure which you need?</h2>
          <p className="prose-body mx-auto mt-3 max-w-prose">
            Most teams start with a readiness review — a fixed-scope look at your infrastructure
            that ends in a prioritized plan you can act on with or without us.
          </p>
          <Link href="/contact" className="btn-primary mt-6">
            Book a readiness review
          </Link>
        </div>
      </section>
    </>
  );
}
