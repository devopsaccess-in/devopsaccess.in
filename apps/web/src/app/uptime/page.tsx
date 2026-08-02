import type { Metadata } from "next";
import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import { softwareApplication } from "@/lib/schema";

export const metadata: Metadata = {
  title: "Uptime monitoring that tells you why — devopsaccess",
  description:
    "Monitor websites, APIs and cron jobs from ₹0 during early access. Alerts name the cause — expired TLS certificate, DNS failure, a backup that never ran — instead of just saying 'down'. IST timestamps, Indian support.",
  alternates: { canonical: "/uptime/" },
};

const APP_URL = "https://app.devopsaccess.in";

// The visible feature copy and the schema featureList are generated from this
// one list, so they cannot drift apart.
const features = [
  {
    name: "Alerts that name the cause",
    body: "Most tools tell you a site is down. We tell you the TLS certificate expired two days ago, or DNS stopped resolving, or the connection was refused — because we record which phase of the request broke.",
  },
  {
    name: "Cron and backup monitoring",
    body: "Your nightly backup pings a secret URL when it finishes. If a run is ever missed, we alert you. This is the failure nobody catches: the job that quietly stopped weeks ago and nobody noticed until it was needed.",
  },
  {
    name: "TLS certificate expiry warnings",
    body: "We watch every certificate we see and warn you 14 days and 3 days before it expires — so the outage never happens, rather than being reported after it does.",
  },
  {
    name: "Where the time goes",
    body: "Every check records DNS, TCP connect, TLS handshake and server time separately, so a site that is technically up but getting slower shows you which part is to blame.",
  },
  {
    name: "Email and Slack alerts, with recovery notices",
    body: "You are told when something breaks and again when it recovers, with how long it was down. Send a test to any channel before you rely on it.",
  },
  {
    name: "Public status page and embeddable badge",
    body: "Turn on a status page your own users can watch, and drop a live uptime badge in your README. Off by default — your monitors are private until you decide otherwise.",
  },
];

export default function Uptime() {
  return (
    <>
      <SiteJsonLd
        pathname="/uptime/"
        extra={[softwareApplication(features.map((f) => f.name))]}
      />

      <section className="container-px pt-16 pb-10">
        <p className="eyebrow">Uptime monitoring</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Know <span className="text-node">why</span> it broke, not just that it did.
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          Uptime monitoring for websites, APIs and the cron jobs everyone forgets. Built for
          Indian startups who are paying for four tools and reading none of them.
        </p>
        <div className="mt-8 flex flex-wrap gap-4">
          <a href={APP_URL} className="btn-primary">
            Start monitoring — free
          </a>
          <Link href="/contact" className="btn-ghost">
            Talk to us first
          </Link>
        </div>
        <p className="mt-4 font-mono text-xs text-mist-faint">
          Free while in early access · no card · sign in with Google
        </p>
      </section>

      <section className="container-px pb-12">
        <div className="grid gap-5 sm:grid-cols-2">
          {features.map((f) => (
            <div key={f.name} className="card">
              <h2 className="text-lg font-semibold text-white">{f.name}</h2>
              <p className="prose-body mt-2 text-sm">{f.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="container-px pb-12">
        <h2 className="text-2xl">What an alert actually looks like</h2>
        <pre className="mt-5 overflow-x-auto rounded-lg border border-ink-line bg-ink-soft p-5 text-sm">
          <code className="text-mist">{`DOWN: checkout API is failing

Monitor: checkout API
URL: https://api.example.in/healthz
Since: 02 Aug 2026 20:51:17 IST
Cause: 2 consecutive failures: TLS certificate expired 2 days ago`}</code>
        </pre>
        <p className="prose-body mt-4 max-w-prose text-sm">
          That last line is the whole point. You know what to fix before you open a laptop.
        </p>
      </section>

      <section className="container-px pb-12">
        <h2 className="text-2xl">Built India-first</h2>
        <div className="mt-5 grid gap-5 sm:grid-cols-3">
          <div className="card">
            <h3 className="font-semibold text-white">IST everywhere</h3>
            <p className="prose-body mt-2 text-sm">
              Every timestamp, in every alert and every page, is Asia/Kolkata. No mental
              arithmetic at 3am.
            </p>
          </div>
          <div className="card">
            <h3 className="font-semibold text-white">Priced in ₹</h3>
            <p className="prose-body mt-2 text-sm">
              Paid plans will be rupee-priced, not a dollar figure converted at whatever
              today&apos;s rate is. Free while we are in early access.
            </p>
          </div>
          <div className="card">
            <h3 className="font-semibold text-white">Support in your timezone</h3>
            <p className="prose-body mt-2 text-sm">
              You email{" "}
              <a href="mailto:support@devopsaccess.in" className="text-node hover:underline">
                support@devopsaccess.in
              </a>{" "}
              and a person who built it replies, during Indian hours.
            </p>
          </div>
        </div>
      </section>

      <section className="container-px pb-20">
        <div className="card">
          <h2 className="text-2xl">Start in about a minute</h2>
          <ol className="prose-body mt-4 max-w-prose list-decimal space-y-2 pl-5 text-sm">
            <li>Sign in with Google — a workspace is created for you.</li>
            <li>Add the address or Slack channel your team actually watches.</li>
            <li>Point a monitor at a URL, or paste a one-line curl into your cron job.</li>
          </ol>
          <a href={APP_URL} className="btn-primary mt-6">
            Start monitoring — free
          </a>
        </div>
      </section>
    </>
  );
}
