import type { Metadata } from "next";
import SiteJsonLd from "@/components/SiteJsonLd";
import ContactForm from "@/components/ContactForm";
import CalendlyEmbed from "@/components/CalendlyEmbed";

export const metadata: Metadata = {
  title: "Contact — devopsaccess",
  description: "Send a message or book a 30-minute discovery call with devopsaccess.",
  alternates: { canonical: "/contact/" },
};

// Calendly free-tier intro/demo call.
const CALENDLY_URL = "https://calendly.com/vikram-devopsaccess/intro-demo-call";
const SUPPORT_EMAIL = "support@devopsaccess.in";

export default function Contact() {
  return (
    <>
      <SiteJsonLd pathname="/contact/" />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Contact</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Let&apos;s look at your setup together.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          Send a note below or grab a time directly. Come as you are — a half-broken pipeline and a
          vague sense of dread are perfectly valid starting points.
        </p>
        <p className="mt-4 font-mono text-xs text-mist-dim">
          Hours: 9 AM – 5 PM IST ·{" "}
          <a href={`mailto:${SUPPORT_EMAIL}`} className="text-node hover:underline">
            {SUPPORT_EMAIL}
          </a>
        </p>
      </section>

      <section className="container-px grid gap-8 pb-16 lg:grid-cols-2">
        {/* Message form */}
        <div>
          <h2 className="text-2xl">Send a message</h2>
          <ContactForm />
        </div>

        {/* Calendly booking */}
        <div>
          <h2 className="text-2xl">Or book a call</h2>
          <div className="mt-6 overflow-hidden rounded-xl border border-ink-line bg-ink-soft">
            <CalendlyEmbed url={CALENDLY_URL} />
            <noscript>
              <div className="p-8 text-center">
                <p className="prose-body">
                  Enable JavaScript to load the scheduler, or email{" "}
                  <a href={`mailto:${SUPPORT_EMAIL}`} className="text-node">
                    {SUPPORT_EMAIL}
                  </a>
                  .
                </p>
              </div>
            </noscript>
          </div>
        </div>
      </section>
    </>
  );
}
