import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";

export const metadata: Metadata = {
  title: "Uses — devopsaccess",
  description: "The exact stack behind this site and the daily toolkit.",
  alternates: { canonical: "/uses/" },
};

const groups = [
  {
    name: "This website",
    items: [
      ["Host", "Hetzner Cloud (cax11, Ubuntu 24.04)"],
      ["Provisioning", "Terraform (hcloud provider)"],
      ["Config management", "Ansible — common, ssh_hardening, fail2ban, nginx, deploy_site"],
      ["Web server", "Nginx, TLS via Cloudflare Origin certificate"],
      ["Edge", "Cloudflare (Full Strict, WAF, CDN)"],
      ["CI/CD", "GitHub Actions — infra on dispatch, site on push"],
      ["Site", "Next.js + Tailwind + MDX, static export"],
    ],
  },
  {
    name: "Daily driver",
    items: [
      ["Clouds", "GCP (GKE), AWS, Hetzner"],
      ["IaC", "Terraform, occasionally Pulumi"],
      ["Containers", "Docker, Kubernetes, Argo CD"],
      ["Observability", "Prometheus, Grafana, Loki"],
      ["Editor", "Neovim / VS Code"],
    ],
  },
];

export default function Uses() {
  return (
    <>
      <SiteJsonLd pathname="/uses/" />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Uses</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Everything here is the pitch.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          This site isn&apos;t hosted on a page builder — it&apos;s a small, real piece of
          infrastructure. Here&apos;s exactly what runs it.
        </p>
      </section>

      <section className="container-px space-y-8 pb-16">
        {groups.map((g) => (
          <div key={g.name}>
            <h2 className="text-2xl">{g.name}</h2>
            <dl className="mt-4 divide-y divide-ink-line border-y border-ink-line">
              {g.items.map(([k, v]) => (
                <div key={k} className="grid grid-cols-1 gap-1 py-3 sm:grid-cols-[12rem_1fr]">
                  <dt className="font-mono text-xs uppercase tracking-wide text-node">{k}</dt>
                  <dd className="text-sm text-mist">{v}</dd>
                </div>
              ))}
            </dl>
          </div>
        ))}
        <p className="prose-body max-w-prose">
          Want the full story? The{" "}
          <Link href="/blog" className="text-node hover:underline">
            blog
          </Link>{" "}
          walks through building this from an empty cloud project to a live site.
        </p>
      </section>
    </>
  );
}
