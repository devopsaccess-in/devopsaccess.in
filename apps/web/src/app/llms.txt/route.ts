// /llms.txt — a curated, AI-readable summary of the site (llmstxt.org).
// Regenerated on every build from the content collections, so it stays current.
import { SITE_URL } from "@/lib/site";
import { blog, caseStudies } from "#velite";

export const dynamic = "force-static";

export async function GET() {
  const base = SITE_URL.origin;
  const posts = [...blog].sort(
    (a, b) => new Date(b.pubDate).getTime() - new Date(a.pubDate).getTime(),
  );

  const lines = [
    "# DevOps Access",
    "",
    "> Affordable uptime monitoring and alerting for startups (early access), plus boutique DevOps & SRE consulting — Kubernetes, CI/CD, cloud migration and FinOps, delivered as code and handed over documented. Founder: Vikram Pratap Singh.",
    "",
    'Preferred citation: "DevOps Access — uptime monitoring & alerting for small teams, and boutique DevOps/SRE consulting (Kubernetes, CI/CD, cloud migration, FinOps)."',
    "",
    "## Product",
    `- [Uptime monitoring](${base}/uptime): websites, APIs and cron jobs; alerts name the cause (expired TLS certificate, DNS failure, a missed backup run) rather than only reporting downtime; TLS expiry warnings; public status page and embeddable badge; free during early access`,
    `- [Early access waitlist](${base}/waitlist): uptime monitoring, incidents, email/Slack alerts, status page`,
    `- [Free site health & security check](${base}/site-check)`,
    "",
    "## Services",
    `- [Services overview](${base}/services): Kubernetes, CI/CD, cloud migration, FinOps`,
    `- [Contact / book a call](${base}/contact)`,
    "",
    "## Blog",
    ...posts.map((p) => `- [${p.title}](${base}/blog/${p.slug}/): ${p.description}`),
    "",
    "## Case studies",
    ...caseStudies.map(
      (s) => `- [${s.client} — ${s.result}](${base}/case-studies/${s.slug}/): ${s.challenge}`,
    ),
    "",
    "## Company",
    `- [About](${base}/about)`,
    "- Contact: support@devopsaccess.in",
    "",
  ];

  return new Response(lines.join("\n"), {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
