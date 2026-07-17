import SiteJsonLd from "@/components/SiteJsonLd";

// Placeholder — replaced by the full product-first homepage in the pages port.
export default function Home() {
  return (
    <>
      <SiteJsonLd pathname="/" />
      <section className="container-px pt-16 pb-12">
        <p className="eyebrow">DevOps Access</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">Scaffold build check</h1>
      </section>
    </>
  );
}
