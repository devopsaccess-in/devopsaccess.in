import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";

export const metadata: Metadata = {
  title: "Thank you — devopsaccess",
  description: "Payment received — thank you.",
  alternates: { canonical: "/thank-you/" },
  robots: { index: false },
};

export default function ThankYou() {
  return (
    <>
      <SiteJsonLd pathname="/thank-you/" />
      <section className="container-px py-24 text-center">
        <p className="eyebrow">Payment received</p>
        <h1 className="mt-3 text-4xl font-bold sm:text-5xl">Thank you — you&apos;re booked.</h1>
        <p className="prose-body mx-auto mt-5 max-w-prose text-lg">
          We&apos;ve received your payment and will email you shortly to confirm the session
          details. A receipt is on its way from Razorpay. Questions? Email{" "}
          <a href="mailto:support@devopsaccess.in" className="text-node hover:underline">
            support@devopsaccess.in
          </a>
          .
        </p>
        <Link href="/" className="btn-ghost mt-8 inline-flex">
          Back to home
        </Link>
      </section>
    </>
  );
}
