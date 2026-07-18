import type { MetadataRoute } from "next";
import { abs } from "@/lib/site";
import { blog, caseStudies } from "#velite";

export const dynamic = "force-static";

// Emitted at build as /sitemap.xml (robots.txt points here). Static pages are
// listed explicitly — a page isn't discoverable unless it's routed anyway.
export default function sitemap(): MetadataRoute.Sitemap {
  const staticPaths = [
    "/",
    "/services/",
    "/case-studies/",
    "/devops-ai/",
    "/blog/",
    "/about/",
    "/uses/",
    "/contact/",
    "/site-check/",
    "/waitlist/",
    "/privacy/",
    "/terms/",
    "/refund/",
    "/cookie-policy/",
  ];

  return [
    ...staticPaths.map((p) => ({ url: abs(p) })),
    ...blog.map((p) => ({
      url: abs(`/blog/${p.slug}/`),
      lastModified: new Date(p.updated ?? p.pubDate),
    })),
    ...caseStudies.map((s) => ({
      url: abs(`/case-studies/${s.slug}/`),
      lastModified: new Date(s.pubDate),
    })),
  ];
}
