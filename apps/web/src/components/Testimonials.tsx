import { testimonials } from "@/data/testimonials";

export default function Testimonials() {
  if (testimonials.length === 0) return null;
  return (
    <section className="container-px py-16">
      <p className="eyebrow">What clients say</p>
      <div className="mt-6 grid gap-5 sm:grid-cols-2">
        {testimonials.map((t) => (
          <figure key={t.name} className="card">
            <blockquote className="prose-body text-lg">&ldquo;{t.quote}&rdquo;</blockquote>
            <figcaption className="mt-4 font-mono text-xs text-mist-dim">
              <span className="text-node">{t.name}</span> · {t.role}
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  );
}
