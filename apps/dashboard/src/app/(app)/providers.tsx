"use client";

import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "@/lib/api";
import type { Me } from "@/lib/types";

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api<Me>("/api/me"),
    staleTime: 5 * 60 * 1000,
  });
}

// Bootstrap blocks the app on the first /api/me call: it provisions the
// user + tenant on first login, and every other endpoint 403s until that
// has happened once.
function Bootstrap({ children }: { children: React.ReactNode }) {
  const me = useMe();
  if (me.isPending) {
    return (
      <div className="flex min-h-screen items-center justify-center font-mono text-sm text-mist-dim">
        setting up your workspace…
      </div>
    );
  }
  if (me.isError) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4">
        <p className="text-alert">Could not load your workspace: {me.error.message}</p>
        <a className="btn-ghost" href="/auth/logout">
          Sign out and retry
        </a>
      </div>
    );
  }
  return children;
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { staleTime: 15_000, refetchInterval: 60_000, retry: 1 },
        },
      }),
  );
  return (
    <QueryClientProvider client={client}>
      <Bootstrap>{children}</Bootstrap>
    </QueryClientProvider>
  );
}
