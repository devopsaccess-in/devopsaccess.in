import Link from "next/link";
import SiteJsonLd from "@/components/SiteJsonLd";
import Testimonials from "@/components/Testimonials";
import Waitlist from "@/components/Waitlist";

// The 3 pains, in plain words a founder feels.
const problems = [
  {
    code: "outage",
    title: "Customers find out first",
    body: "The site goes down and the first signal is a support email — or a tweet. By then the damage is done.",
  },
  {
    code: "3am",
    title: "Things break at 3am",
    body: "Something falls over at the worst possible time, and on-call is whoever happens to be awake.",
  },
  {
    code: "bill",
    title: "Monitoring tools cost like a salary",
    body: "Datadog + PagerDuty + a status page adds up to more than a junior engineer. For a small team that math never works.",
  },
];

// The 3 outcomes (not features) — what it's actually worth to them.
const outcomes = [
  {
    metric: "Catch it first",
    body: "Checks every minute from the moment you add a monitor. You know before your customers do.",
  },
  {
    metric: "Alerts that reach you",
    body: "Email and Slack the moment something goes down — and a recovery note when it's back.",
  },
  {
    metric: "A price that makes sense",
    body: "One affordable product instead of three stitched-together subscriptions. Built for small teams.",
  },
];

export default function Home() {
  return (
    <>
      <SiteJsonLd pathname="/" />

      {/* HERO */}
      <section className="container-px pt-16 pb-20 sm:pt-24">
        <div className="status-pill mb-6">
          <span className="status-dot"></span>
          early access · building in the open
        </div>

        <h1 className="max-w-4xl text-4xl font-bold leading-[1.05] sm:text-6xl">
          Know your site is down —<span className="text-node"> before your customers do.</span>
        </h1>

        <p className="prose-body mt-6 max-w-prose text-lg">
          We&apos;re building uptime monitoring and alerting a small team can actually afford:
          minute-by-minute checks, incidents, email and Slack alerts, and a public status page —
          in one product. Join early access to get in first and help shape it.
        </p>

        <div className="mt-9 flex flex-wrap gap-4">
          <Link href="#waitlist" className="btn-primary">
            Join early access
          </Link>
          <Link href="/site-check" className="btn-ghost">
            Try the free site check
          </Link>
        </div>

        {/* SIGNATURE: monitoring panel */}
        <div className="mt-16 overflow-hidden rounded-xl border border-ink-line bg-ink-soft/80">
          <div className="flex items-center gap-2 border-b border-ink-line px-4 py-2.5">
            <span className="h-3 w-3 rounded-full bg-signal/70"></span>
            <span className="h-3 w-3 rounded-full bg-mist-dim/40"></span>
            <span className="h-3 w-3 rounded-full bg-node/70"></span>
            <span className="ml-3 font-mono text-xs text-mist-dim">monitors — last 24h</span>
          </div>
          <div className="overflow-x-auto p-4">
            <table className="w-full min-w-[34rem] border-collapse font-mono text-xs">
              <thead>
                <tr className="text-left text-mist-dim">
                  <th className="pb-2 pr-6 font-medium">MONITOR</th>
                  <th className="pb-2 pr-6 font-medium">STATUS</th>
                  <th className="pb-2 pr-6 font-medium">UPTIME</th>
                  <th className="pb-2 font-medium">LAST INCIDENT</th>
                </tr>
              </thead>
              <tbody className="text-mist">
                <tr className="border-t border-ink-line">
                  <td className="py-2 pr-6">api.yourstartup.com</td>
                  <td className="py-2 pr-6 text-node">Up</td>
                  <td className="py-2 pr-6">100%</td>
                  <td className="py-2">—</td>
                </tr>
                <tr className="border-t border-ink-line">
                  <td className="py-2 pr-6">app.yourstartup.com</td>
                  <td className="py-2 pr-6 text-node">Up</td>
                  <td className="py-2 pr-6">99.98%</td>
                  <td className="py-2">recovered in 4m · alerted on Slack</td>
                </tr>
                <tr className="border-t border-ink-line">
                  <td className="py-2 pr-6">your-site.com</td>
                  <td className="py-2 pr-6 text-signal">Pending</td>
                  <td className="py-2 pr-6">?</td>
                  <td className="py-2">join early access →</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* THE PROBLEM */}
      <section className="container-px py-8">
        <p className="eyebrow">Sound familiar?</p>
        <h2 className="mt-2 text-3xl font-bold sm:text-4xl">
          Downtime you hear about too late, tools that cost too much.
        </h2>
        <div className="mt-10 grid gap-5 sm:grid-cols-3">
          {problems.map((p) => (
            <div key={p.code} className="card">
              <div className="font-mono text-xs text-node">{p.code}</div>
              <h3 className="mt-2 text-xl">{p.title}</h3>
              <p className="prose-body mt-2 text-sm">{p.body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* WHAT YOU GET */}
      <section className="container-px py-8">
        <p className="eyebrow">What it&apos;s worth to you</p>
        <h2 className="mt-2 text-3xl font-bold sm:text-4xl">
          Less downtime. Less spend. More sleep.
        </h2>
        <div className="mt-10 grid gap-6 sm:grid-cols-3">
          {outcomes.map((o) => (
            <div key={o.metric} className="rounded-xl border border-ink-line bg-ink-card/40 p-6">
              <div className="font-display text-2xl text-node">{o.metric}</div>
              <p className="prose-body mt-2 text-sm">{o.body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* FREE TOOL TEASER */}
      <section className="container-px py-8">
        <div className="rounded-xl border border-node/30 bg-node/[0.04] p-8 sm:flex sm:items-center sm:justify-between">
          <div className="max-w-prose">
            <p className="eyebrow">Free · live now</p>
            <h2 className="mt-2 text-2xl">Free Site Health &amp; Security check</h2>
            <p className="prose-body mt-2">
              Instant grade for your security headers, SSL/TLS and SEO basics — for any site you
              own. No signup wall.
            </p>
          </div>
          <Link href="/site-check" className="btn-primary mt-5 shrink-0 sm:mt-0">
            Run a free check
          </Link>
        </div>
      </section>

      {/* PROOF */}
      <Testimonials />

      {/* EARLY ACCESS */}
      <div id="waitlist">
        <Waitlist
          heading="Be first — and help shape it"
          blurb="We're building affordable uptime monitoring in the open. Join early access to try it before launch, lock in early pricing, and tell us what to build next."
        />
      </div>

      {/* SECONDARY: consulting (side door) */}
      <section className="container-px pb-12">
        <p className="prose-body text-center text-sm text-mist-dim">
          Need hands-on DevOps help right now? We take a small number of{" "}
          <Link href="/services" className="text-node hover:underline">
            consulting engagements
          </Link>{" "}
          each month.
        </p>
      </section>
    </>
  );
}
