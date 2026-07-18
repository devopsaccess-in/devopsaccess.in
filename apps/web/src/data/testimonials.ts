// Real testimonials only. Add entries as clients agree to be quoted; the
// Testimonials component hides itself entirely when this list is empty.
export interface Testimonial {
  quote: string;
  name: string;
  role: string; // e.g. "CTO, Acme (fintech)"
}

export const testimonials: Testimonial[] = [
  // {
  //   quote: "They put our infra under version control and our on-call to sleep.",
  //   name: "Jane Doe",
  //   role: "CTO, Example (Series A fintech)",
  // },
];
