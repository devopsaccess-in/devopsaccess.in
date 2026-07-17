// Sitewide constants. The canonical origin everything (metadata, JSON-LD,
// sitemap, llms.txt) derives absolute URLs from.
export const SITE_URL = new URL("https://devopsaccess.in");
export const SITE_NAME = "DevOps Access";

export const DEFAULT_TITLE = "DevOps Access — uptime monitoring & alerting for small teams";
export const DEFAULT_DESCRIPTION =
  "One affordable product for uptime monitoring, incidents and alerting — built for startups that can't justify a Datadog bill. Join early access.";

export const abs = (path: string): string => new URL(path, SITE_URL).href;
