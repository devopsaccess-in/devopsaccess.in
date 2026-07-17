import type { Metadata } from "next";
import SiteJsonLd from "@/components/SiteJsonLd";
import Waitlist from "@/components/Waitlist";

export const metadata: Metadata = {
  title: "Early access — devopsaccess",
  description: "Join the early-access list for affordable uptime monitoring and alerting.",
  alternates: { canonical: "/waitlist/" },
};

export default function WaitlistPage() {
  return (
    <>
      <SiteJsonLd pathname="/waitlist/" />
      <section className="container-px pt-16">
        <p className="eyebrow">Waitlist</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">Get early access.</h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          We&apos;re building uptime monitoring and alerting for teams that have outgrown
          &ldquo;the founder runs prod&rdquo; — without the per-host billing surprises. Leave your
          email and we&apos;ll reach out as we open it up.
        </p>
      </section>
      <Waitlist />
    </>
  );
}
