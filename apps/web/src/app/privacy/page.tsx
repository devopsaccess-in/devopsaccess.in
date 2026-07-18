import type { Metadata } from "next";
import Link from "next/link";
import LegalPage from "@/components/LegalPage";

export const metadata: Metadata = {
  title: "Privacy Policy — devopsaccess",
  description:
    "How devopsaccess collects, uses, and protects personal data, aligned to India's DPDP Act 2023 and the EU GDPR.",
  alternates: { canonical: "/privacy/" },
};

export default function Privacy() {
  return (
    <LegalPage title="Privacy Policy" updated="19 June 2026" pathname="/privacy/">
      <p>
        This Privacy Policy explains how <strong>DevOps Access</strong> (&ldquo;we&rdquo;,
        &ldquo;us&rdquo;), operating <a href="https://devopsaccess.in">devopsaccess.in</a> from
        Bengaluru, India, collects and processes personal data. We act as the data{" "}
        <em>controller</em> for the information described below. We aim to meet the standards of
        India&apos;s <strong>Digital Personal Data Protection Act, 2023 (DPDP)</strong> and the EU{" "}
        <strong>General Data Protection Regulation (GDPR)</strong>.
      </p>

      <h2>1. What we collect</h2>
      <ul>
        <li>
          <strong>Contact submissions</strong> — the name, email address, and message you send via
          our contact form.
        </li>
        <li>
          <strong>Scheduling data</strong> — if you book a call, Calendly collects the details you
          provide (handled under Calendly&apos;s own policy).
        </li>
        <li>
          <strong>Usage analytics</strong> — if you accept the Analytics category: pages viewed,
          time on page, referrer/traffic source, approximate location and browser, and masked
          session recordings (PostHog, processed in the US), plus aggregate measurement (Google
          Analytics). Cloudflare Web Analytics is cookieless and always on. See our{" "}
          <Link href="/cookie-policy">Cookie Policy</Link>; change your choice anytime via the
          footer&apos;s &ldquo;cookie preferences&rdquo;.
        </li>
      </ul>

      <h2>2. Why we use it (lawful basis)</h2>
      <ul>
        <li>
          To respond to your enquiries and provide our services — performance of a contract / your
          request.
        </li>
        <li>
          To understand site performance and improve content — our legitimate interests, balanced
          against your rights.
        </li>
        <li>
          Where required, on the basis of your <strong>consent</strong>, which you may withdraw at
          any time.
        </li>
      </ul>

      <h2>3. How long we keep it</h2>
      <p>
        Contact messages are retained only as long as needed to handle your enquiry and our ongoing
        relationship, after which they are deleted. Analytics data is aggregated and not tied to an
        identifiable person.
      </p>

      <h2>4. Sharing and processors</h2>
      <p>
        We do not sell your personal data. We use a small set of processors to operate the site:
        Cloudflare (CDN, edge analytics, bot protection), Google Workspace (email), Calendly
        (scheduling), Razorpay (payments), and — only with your consent — Google Analytics and
        PostHog (analytics). Each processes data under its own terms and appropriate safeguards for
        international transfers.
      </p>

      <h2>5. Your rights</h2>
      <p>Subject to applicable law, you may request to:</p>
      <ul>
        <li>access, correct, or erase your personal data;</li>
        <li>restrict or object to processing, and withdraw consent;</li>
        <li>receive a copy of data you provided (portability), where applicable;</li>
        <li>nominate, under DPDP, another person to exercise your rights;</li>
        <li>
          lodge a complaint with the relevant Data Protection Board / supervisory authority.
        </li>
      </ul>

      <h2>6. International transfers (EU/EEA)</h2>
      <p>
        Where data is processed outside the EEA, we rely on appropriate safeguards (such as
        Standard Contractual Clauses) provided by our processors to protect your information to
        European standards.
      </p>

      <h2>7. Contact</h2>
      <p>
        For any privacy request or question, email{" "}
        <a href="mailto:support@devopsaccess.in">support@devopsaccess.in</a>. We will respond
        within the timelines required by applicable law.
      </p>
    </LegalPage>
  );
}
