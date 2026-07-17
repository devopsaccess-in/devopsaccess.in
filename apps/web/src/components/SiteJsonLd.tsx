import { organization, website, person, breadcrumbs } from "@/lib/schema";

// Single JSON-LD @graph per page: sitewide entities (Organization, WebSite,
// founder Person) + auto breadcrumb + page-specific nodes. Rendered per page
// because a static-export layout cannot know the current pathname.
export default function SiteJsonLd({
  pathname,
  extra = [],
}: {
  pathname: string;
  extra?: object[];
}) {
  const jsonLd = {
    "@context": "https://schema.org",
    "@graph": [organization(), website(), person(), ...breadcrumbs(pathname), ...extra],
  };
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
    />
  );
}
