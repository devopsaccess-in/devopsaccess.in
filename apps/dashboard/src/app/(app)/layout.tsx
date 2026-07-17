import { Providers } from "./providers";
import { Nav } from "@/components/nav";

// Authenticated app shell: middleware guarantees a session for everything in
// this group; Providers bootstraps /api/me (first-login provisioning).
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <Providers>
      <div className="min-h-screen">
        <Nav />
        <main className="container-px py-8">{children}</main>
      </div>
    </Providers>
  );
}
