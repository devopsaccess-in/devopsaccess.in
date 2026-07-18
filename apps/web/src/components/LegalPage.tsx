import SiteJsonLd from "@/components/SiteJsonLd";

// Wrapper for legal pages: eyebrow, title, last-updated line, and typographic
// defaults for the body (the .legal styles live in globals.css).
export default function LegalPage({
  title,
  updated,
  pathname,
  children,
}: {
  title: string;
  updated: string; // human-readable date
  pathname: string;
  children: React.ReactNode;
}) {
  return (
    <>
      <SiteJsonLd pathname={pathname} />
      <section className="container-px pt-16 pb-16">
        <p className="eyebrow">Legal</p>
        <h1 className="mt-2 text-4xl font-bold sm:text-5xl">{title}</h1>
        <p className="mt-3 font-mono text-xs text-mist-dim">Last updated: {updated}</p>

        <div className="legal mt-8 max-w-prose">{children}</div>
      </section>
    </>
  );
}
