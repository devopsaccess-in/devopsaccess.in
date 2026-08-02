// JSON-LD schema.org node builders. Each page composes these into a single
// @graph (via <SiteJsonLd>). Stable @id fragments let nodes reference each other.

import { SITE_URL, abs } from "@/lib/site";

type Node = Record<string, unknown>;

export function organization(): Node {
  return {
    "@type": "Organization",
    "@id": abs("/#organization"),
    name: "DevOps Access",
    url: SITE_URL.href,
    email: "support@devopsaccess.in",
    description:
      "Uptime monitoring, incidents and alerting in one affordable product for startups — plus hands-on DevOps expertise behind it. India-built, works anywhere.",
    foundingDate: "2026",
    areaServed: "IN",
    knowsAbout: [
      "Uptime Monitoring",
      "Alerting",
      "DevOps",
      "SRE",
      "Kubernetes",
      "CI/CD",
      "FinOps",
      "Observability",
    ],
    founder: { "@id": abs("/#founder") },
    contactPoint: {
      "@type": "ContactPoint",
      email: "support@devopsaccess.in",
      contactType: "customer support",
      availableLanguage: ["English", "Hindi"],
    },
    sameAs: [
      "https://linkedin.com/company/devopsaccess",
      "https://x.com/devopsaccess",
      "https://github.com/devopsaccess-in",
    ],
  };
}

export function website(): Node {
  return {
    "@type": "WebSite",
    "@id": abs("/#website"),
    url: SITE_URL.href,
    name: "DevOps Access",
    publisher: { "@id": abs("/#organization") },
  };
}

export function person(): Node {
  return {
    "@type": "Person",
    "@id": abs("/#founder"),
    name: "Vikram Pratap Singh",
    jobTitle: "Founder",
    url: abs("/about"),
    worksFor: { "@id": abs("/#organization") },
    knowsAbout: ["Kubernetes", "GKE", "GCP", "AWS", "SRE", "FinOps", "Terraform"],
    sameAs: ["https://linkedin.com/company/devopsaccess", "https://github.com/devopsaccess-in"],
  };
}

// Auto breadcrumb from the URL path. Returns [] for the homepage.
export function breadcrumbs(pathname: string): Node[] {
  const parts = pathname.split("/").filter(Boolean);
  if (parts.length === 0) return [];
  const items = [{ "@type": "ListItem", position: 1, name: "Home", item: SITE_URL.href }];
  let acc = "";
  parts.forEach((p, i) => {
    acc += `/${p}`;
    items.push({
      "@type": "ListItem",
      position: i + 2,
      name: p.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()),
      item: abs(acc + "/"),
    });
  });
  return [{ "@type": "BreadcrumbList", itemListElement: items }];
}

export function blogPosting(
  url: string,
  data: { title: string; description: string; pubDate: Date; updated?: Date },
): Node {
  return {
    "@type": "BlogPosting",
    headline: data.title,
    description: data.description,
    datePublished: data.pubDate.toISOString(),
    dateModified: (data.updated ?? data.pubDate).toISOString(),
    author: { "@id": abs("/#founder") },
    publisher: { "@id": abs("/#organization") },
    mainEntityOfPage: url,
    image: abs("/og.png"),
  };
}

export function caseStudyArticle(
  url: string,
  data: { client: string; challenge: string; result: string; pubDate: Date },
): Node {
  return {
    "@type": "Article",
    headline: `${data.client} — ${data.result}`,
    description: data.challenge,
    datePublished: data.pubDate.toISOString(),
    dateModified: data.pubDate.toISOString(),
    author: { "@id": abs("/#founder") },
    publisher: { "@id": abs("/#organization") },
    mainEntityOfPage: url,
  };
}

export function serviceList(services: { title: string; summary: string }[]): Node {
  return {
    "@type": "ItemList",
    name: "DevOps Access services",
    itemListElement: services.map((s, i) => ({
      "@type": "ListItem",
      position: i + 1,
      item: {
        "@type": "Service",
        name: s.title,
        description: s.summary,
        provider: { "@id": abs("/#organization") },
        areaServed: "IN",
      },
    })),
  };
}

export function itemList(
  name: string,
  items: { name: string; url: string; description: string }[],
): Node {
  return {
    "@type": "ItemList",
    name,
    itemListElement: items.map((it, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: it.name,
      url: it.url,
      description: it.description,
    })),
  };
}

// The uptime product itself. Every claim here has to be visible on the page
// that emits it — Google penalises schema that overstates the rendered
// content, and the featureList is the easiest place to drift.
export function softwareApplication(featureList: string[]): Node {
  return {
    "@type": "SoftwareApplication",
    "@id": abs("/uptime/#software"),
    name: "DevOps Access Uptime",
    url: abs("/uptime/"),
    applicationCategory: "DeveloperApplication",
    applicationSubCategory: "Website Monitoring",
    operatingSystem: "Web",
    description:
      "Uptime monitoring for websites, APIs and cron jobs, with alerts that say why something broke — expired TLS certificates, DNS failures, missed backup runs — not just that it did.",
    provider: { "@id": abs("/#organization") },
    inLanguage: "en",
    featureList,
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "INR",
      availability: "https://schema.org/LimitedAvailability",
      description: "Free during early access",
    },
  };
}
