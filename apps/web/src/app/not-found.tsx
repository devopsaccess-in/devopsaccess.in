import Link from "next/link";

export default function NotFound() {
  return (
    <section className="container-px pt-24 pb-16 text-center">
      <p className="eyebrow">404</p>
      <h1 className="mt-2 text-4xl font-bold sm:text-5xl">Page not found</h1>
      <p className="prose-body mx-auto mt-4 max-w-prose">
        The page you&apos;re looking for doesn&apos;t exist or has moved.
      </p>
      <div className="mt-8">
        <Link href="/" className="btn-primary">
          Back to home
        </Link>
      </div>
    </section>
  );
}
