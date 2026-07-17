"use client";

import Link from "next/link";
import { track } from "@/lib/track";

// Razorpay Payment Link. The URL is public and injected at build from
// NEXT_PUBLIC_RAZORPAY_PAYMENT_URL. Payment happens on Razorpay's hosted page,
// so the site loads no Razorpay JS/cookies. Empty => graceful fallback to contact.
const PAYMENT_URL = process.env.NEXT_PUBLIC_RAZORPAY_PAYMENT_URL || "";

export default function PaymentButton() {
  const onClick = () => track("payment_cta_clicked");
  return PAYMENT_URL ? (
    <a href={PAYMENT_URL} target="_blank" rel="noopener" className="btn-primary" onClick={onClick}>
      Book a paid session
    </a>
  ) : (
    <Link href="/contact" className="btn-ghost" onClick={onClick}>
      Enquire about a paid session
    </Link>
  );
}
