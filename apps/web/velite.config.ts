import { defineConfig, defineCollection, s } from "velite";

// Content collections — the Next.js equivalent of Astro's src/content/config.ts.
// Velite validates frontmatter with Zod-style schemas at build time and emits
// typed data + compiled HTML into .velite/ (imported via the "#velite" alias).

const blog = defineCollection({
  name: "Blog",
  pattern: "blog/**/*.mdx",
  schema: s
    .object({
      title: s.string(),
      description: s.string(),
      pubDate: s.isodate(),
      updated: s.isodate().optional(),
      tags: s.array(s.string()).default([]),
      readingTime: s.string().optional(),
      path: s.path(),
      html: s.markdown(),
      raw: s.raw(),
    })
    .transform((data) => ({ ...data, slug: data.path.replace(/^blog\//, "") })),
});

const caseStudies = defineCollection({
  name: "CaseStudy",
  pattern: "case-studies/**/*.mdx",
  schema: s
    .object({
      client: s.string(),
      sector: s.string(),
      challenge: s.string(),
      stack: s.array(s.string()).default([]),
      result: s.string(),
      pubDate: s.isodate(),
      path: s.path(),
      html: s.markdown(),
      raw: s.raw(),
    })
    .transform((data) => ({ ...data, slug: data.path.replace(/^case-studies\//, "") })),
});

export default defineConfig({
  root: "content",
  collections: { blog, caseStudies },
});
