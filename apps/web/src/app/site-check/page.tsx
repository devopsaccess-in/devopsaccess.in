import type { Metadata } from "next";
import SiteJsonLd from "@/components/SiteJsonLd";
import SiteCheck from "@/components/SiteCheck";

export const metadata: Metadata = {
  title: "Free Site Health & Security check — devopsaccess",
  description:
    "Free, instant report on your site's security headers, SSL/TLS, and SEO basics. No signup wall.",
  alternates: { canonical: "/site-check/" },
};

export default function SiteCheckPage() {
  return (
    <>
      <SiteJsonLd pathname="/site-check/" />
      <SiteCheck />
    </>
  );
}
