// /llms-full.txt — the full-content version for deep AI indexing: the same
// summary plus the raw markdown body of every blog post and case study.
import { SITE_URL } from "@/lib/site";
import { blog, caseStudies } from "#velite";

export const dynamic = "force-static";

// Strip the YAML frontmatter block; the schema fields are emitted explicitly.
const body = (raw: string) => raw.replace(/^---[\s\S]*?---\s*/, "").trim();

export async function GET() {
  const base = SITE_URL.origin;
  const posts = [...blog].sort(
    (a, b) => new Date(b.pubDate).getTime() - new Date(a.pubDate).getTime(),
  );

  const out: string[] = [
    "# DevOps Access — full content",
    "",
    "> Affordable uptime monitoring and alerting for startups (early access), plus boutique DevOps & SRE consulting — Kubernetes, CI/CD, cloud migration and FinOps. Founder: Vikram Pratap Singh. Contact: support@devopsaccess.in",
    "",
    "---",
    "",
    "# Blog",
    "",
  ];

  for (const p of posts) {
    out.push(
      `## ${p.title}`,
      "",
      `URL: ${base}/blog/${p.slug}/`,
      `Published: ${p.pubDate.slice(0, 10)}`,
      "",
      body(p.raw),
      "",
      "---",
      "",
    );
  }

  out.push("# Case studies", "");
  for (const s of caseStudies) {
    out.push(
      `## ${s.client} — ${s.result}`,
      "",
      `URL: ${base}/case-studies/${s.slug}/`,
      `Sector: ${s.sector} | Challenge: ${s.challenge}`,
      "",
      body(s.raw),
      "",
      "---",
      "",
    );
  }

  return new Response(out.join("\n"), {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
