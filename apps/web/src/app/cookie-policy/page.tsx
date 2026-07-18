import type { Metadata } from "next";
import Link from "next/link";
import LegalPage from "@/components/LegalPage";

export const metadata: Metadata = {
  title: "Cookie Policy — devopsaccess",
  description:
    "How devopsaccess uses cookies and cookieless analytics, and the choices available to you.",
  alternates: { canonical: "/cookie-policy/" },
};

export default function CookiePolicy() {
  return (
    <LegalPage title="Cookie Policy" updated="19 June 2026" pathname="/cookie-policy/">
      <p>
        This Cookie Policy explains how <strong>DevOps Access</strong> uses cookies and similar
        technologies on <a href="https://devopsaccess.in">devopsaccess.in</a>.
      </p>

      <h2>1. Categories you control</h2>
      <p>
        Our consent banner lets you allow or reject each category below; you can change your choice
        anytime via the <strong>&ldquo;cookie preferences&rdquo;</strong> link in the footer.
      </p>
      <ul>
        <li>
          <strong>Strictly necessary</strong> — <em>always on.</em> Stores your consent choice
          locally and powers the contact form. Not used for tracking.
        </li>
        <li>
          <strong>Analytics</strong> — <em>opt-in.</em> Loads only after you accept (see section
          2).
        </li>
      </ul>
      <p className="mt-3">
        Separately, <strong>Cloudflare Web Analytics</strong> measures aggregate, anonymous traffic
        at the edge with <em>no cookies</em> and no cross-site tracking — it needs no consent. We
        honour browser &ldquo;Do Not Track&rdquo; / Global Privacy Control signals where supported.
      </p>

      <h2>2. What the Analytics category collects</h2>
      <p>If you accept Analytics, we load:</p>
      <ul>
        <li>
          <strong>PostHog</strong> (product analytics, browser local storage — no cookies): pages
          viewed, time on page, referrer/traffic source, approximate location and browser, and{" "}
          <strong>session recordings</strong> with all form inputs masked (we never record what you
          type). Data is processed in the United States under appropriate safeguards.
        </li>
        <li>
          <strong>Google Analytics (GA4)</strong>: aggregate audience/traffic measurement. GA4{" "}
          <strong>sets cookies</strong> (e.g. <code>_ga</code>, <code>_ga_*</code>) holding a
          random identifier, not your personal details.
        </li>
      </ul>
      <p>Reject Analytics and neither PostHog nor GA4 loads, and no session is recorded.</p>

      <h2>3. Third-party embeds</h2>
      <p>
        If you book a call, the embedded <strong>Calendly</strong> scheduler may set its own
        cookies under Calendly&apos;s policy. These load only on the contact page when you interact
        with the scheduler.
      </p>

      <h2>4. Your choices</h2>
      <ul>
        <li>
          On the banner: <strong>Accept all</strong>, <strong>Reject all</strong>, or{" "}
          <strong>Manage</strong> to toggle the Analytics category.
        </li>
        <li>
          Re-open the chooser anytime via <strong>&ldquo;cookie preferences&rdquo;</strong> in the
          footer.
        </li>
        <li>You can also clear local storage / cookies in your browser to reset your choice.</li>
      </ul>

      <h2>5. Contact</h2>
      <p>
        Questions? Email <a href="mailto:support@devopsaccess.in">support@devopsaccess.in</a>. See
        also our <Link href="/privacy">Privacy Policy</Link>.
      </p>
    </LegalPage>
  );
}
