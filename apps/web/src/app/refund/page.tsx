import type { Metadata } from "next";
import LegalPage from "@/components/LegalPage";

export const metadata: Metadata = {
  title: "Refund Policy — devopsaccess",
  description: "Refund and cancellation terms for devopsaccess engagements.",
  alternates: { canonical: "/refund/" },
};

export default function Refund() {
  return (
    <LegalPage title="Refund Policy" updated="19 June 2026" pathname="/refund/">
      <p>
        This Refund Policy applies to services purchased from <strong>DevOps Access</strong>{" "}
        (&ldquo;we&rdquo;, &ldquo;us&rdquo;). Specific terms for a given engagement are set out in
        its proposal or statement of work and prevail over this general policy.
      </p>

      <h2>1. Consulting and engineering engagements</h2>
      <p>
        Work is typically billed against an agreed scope, milestones, or time. Fees for work
        already performed and accepted are non-refundable. If you are not satisfied with a
        deliverable, tell us within <strong>7 days</strong> of delivery and we will work in good
        faith to correct it.
      </p>

      <h2>2. Cancellations</h2>
      <ul>
        <li>
          You may cancel an engagement with written notice; you remain responsible for work
          completed and committed costs up to the cancellation date.
        </li>
        <li>
          Pre-paid amounts for work not yet started are refundable, less any non-recoverable
          third-party costs.
        </li>
      </ul>

      <h2>3. Discovery calls</h2>
      <p>
        Initial discovery calls are free. Bookings can be rescheduled or cancelled via the
        confirmation email at no charge.
      </p>

      <h2>4. How refunds are issued</h2>
      <p>
        Approved refunds are returned to the original payment method within a reasonable period,
        subject to your bank or payment provider&apos;s processing times.
      </p>

      <h2>5. Contact</h2>
      <p>
        To request a refund or discuss a concern, email{" "}
        <a href="mailto:support@devopsaccess.in">support@devopsaccess.in</a>.
      </p>
    </LegalPage>
  );
}
