import type { Metadata } from "next";
import LegalPage from "@/components/LegalPage";

export const metadata: Metadata = {
  title: "Terms & Conditions — devopsaccess",
  description:
    "The terms governing your use of the devopsaccess website and engagement of our services.",
  alternates: { canonical: "/terms/" },
};

export default function Terms() {
  return (
    <LegalPage title="Terms & Conditions" updated="19 June 2026" pathname="/terms/">
      <p>
        These Terms govern your use of <a href="https://devopsaccess.in">devopsaccess.in</a> and
        any services provided by <strong>DevOps Access</strong> (&ldquo;we&rdquo;,
        &ldquo;us&rdquo;). By using this site or engaging us, you agree to these Terms.
      </p>

      <h2>1. Services</h2>
      <p>
        We provide DevOps, Kubernetes, cloud-migration, and FinOps consulting and engineering
        services. The specific scope, deliverables, fees, and timelines of any engagement are set
        out in a separate written proposal or statement of work, which prevails over general
        descriptions on this site.
      </p>

      <h2>2. Use of the site</h2>
      <ul>
        <li>You may not use the site unlawfully or attempt to disrupt or compromise it.</li>
        <li>
          Content on this site is provided for general information and may change without notice.
        </li>
        <li>
          All trademarks and content are owned by us or our licensors unless stated otherwise.
        </li>
      </ul>

      <h2>3. Engagements and payment</h2>
      <p>
        Fees, invoicing, and payment terms are defined in the applicable proposal or statement of
        work. Unless agreed otherwise, invoices are payable within the period stated on the
        invoice.
      </p>

      <h2>4. Confidentiality</h2>
      <p>
        We treat non-public information you share in the course of an engagement as confidential
        and use it only to deliver the agreed services.
      </p>

      <h2>5. Warranties and liability</h2>
      <p>
        Services are provided with reasonable skill and care. To the maximum extent permitted by
        law, the site and its content are provided &ldquo;as is&rdquo;, and our aggregate liability
        arising from an engagement is limited to the fees paid for that engagement. We are not
        liable for indirect or consequential loss.
      </p>

      <h2>6. Governing law</h2>
      <p>
        These Terms are governed by the laws of India, and the courts of Bengaluru, Karnataka have
        exclusive jurisdiction, without prejudice to any mandatory consumer protections available
        to you in your place of residence.
      </p>

      <h2>7. Contact</h2>
      <p>
        Questions about these Terms? Email{" "}
        <a href="mailto:support@devopsaccess.in">support@devopsaccess.in</a>.
      </p>
    </LegalPage>
  );
}
