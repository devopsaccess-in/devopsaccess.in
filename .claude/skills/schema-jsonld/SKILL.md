---
name: schema-jsonld
description: Use this skill whenever the user asks to add, modify, or validate JSON-LD structured data / schema.org markup on devopsaccess.in pages. Covers Organization, WebSite, BreadcrumbList, Product, FAQ, Article, HowTo, Event, and Person schemas. Triggers on phrases like "schema markup", "JSON-LD", "structured data", "rich results", "SEO schema", "search schema".
---

# Schema Markup — JSON-LD Patterns for DevOps Access

See the detailed Schema Markup Strategy page in Notion for full context. This skill is the implementation quick reference.

## Core principles

- JSON-LD only (Google's recommended format). No Microdata or RDFa.
- Inject via SSR (never client-side). Next.js `<script>` tag inside the component's JSX return.
- Use `@graph` arrays to combine multiple types on one page.
- Use stable `@id` values so entities reference each other across pages.
- Auto-generate from the same data source that renders the visible page.

## Reusable JsonLd component

```tsx
// apps/web/src/components/JsonLd.tsx
export function JsonLd({ data }: { data: object }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        // Escape < to prevent XSS via closing script tag
        __html: JSON.stringify(data).replace(/</g, '\\u003c'),
      }}
    />
  );
}
```

## Sitewide Organization + WebSite (inject in root layout)

```typescript
// apps/web/src/lib/schema/sitewide.ts
export const sitewideSchema = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': 'https://devopsaccess.in/#organization',
      name: 'DevOps Access',
      url: 'https://devopsaccess.in',
      logo: {
        '@type': 'ImageObject',
        url: 'https://devopsaccess.in/logo.png',
        width: 300,
        height: 300,
      },
      description:
        'Unified DevOps lifecycle platform — hosting, monitoring, alerting, APM, on-call paging, CI/CD. Built for Indian startups.',
      foundingDate: '2026',
      founder: {
        '@type': 'Person',
        '@id': 'https://devopsaccess.in/#founder',
        name: 'Vikram Pratap Singh',
      },
      contactPoint: {
        '@type': 'ContactPoint',
        email: 'support@devopsaccess.in',
        contactType: 'customer support',
        availableLanguage: ['English', 'Hindi'],
      },
      sameAs: [
        'https://twitter.com/devopsaccess',
        'https://linkedin.com/company/devopsaccess',
        'https://github.com/devopsaccess',
      ],
      knowsAbout: [
        'DevOps', 'SRE', 'Kubernetes', 'Observability', 'APM', 'On-Call Management',
      ],
    },
    {
      '@type': 'WebSite',
      '@id': 'https://devopsaccess.in/#website',
      name: 'DevOps Access',
      url: 'https://devopsaccess.in',
      publisher: { '@id': 'https://devopsaccess.in/#organization' },
      potentialAction: {
        '@type': 'SearchAction',
        target: 'https://devopsaccess.in/search?q={search_term_string}',
        'query-input': 'required name=search_term_string',
      },
    },
  ],
};
```

## Root layout injection

```tsx
// apps/web/src/app/layout.tsx
import { JsonLd } from '@/components/JsonLd';
import { sitewideSchema } from '@/lib/schema/sitewide';

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <JsonLd data={sitewideSchema} />
      </head>
      <body>{children}</body>
    </html>
  );
}
```

## Hard rules

1. Schema content MUST match visible page content exactly (Google penalizes mismatches).
2. `dateModified` must auto-update from git commit timestamps, not hardcoded.
3. Do NOT add FAQPage schema to blog posts (Google March 2026 penalty — only on dedicated FAQ pages).
4. Keep FAQ answers under 80 words for best AI citation.
5. One JSON-LD block per page using `@graph`; don't emit multiple `<script>` tags.
6. Validate with both Google Rich Results Test and Schema.org Validator after deploy.

## Validation commands

```bash
# Local dev: inspect what's on the page
curl -s http://localhost:3000/pricing | grep -A 500 'application/ld+json'

# Production (Google Rich Results Test)
open "https://search.google.com/test/rich-results?url=https://devopsaccess.in/pricing"

# Production (Schema.org validator, more thorough)
open "https://validator.schema.org?url=https://devopsaccess.in/pricing"
```

For full reference on Product, Article, HowTo, Event, Person, BreadcrumbList schemas, see the Schema Markup Strategy page in Notion.
