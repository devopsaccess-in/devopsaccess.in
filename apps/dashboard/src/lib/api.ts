"use client";

import { getAccessToken } from "@auth0/nextjs-auth0";

// api calls the Go control plane same-origin (nginx proxies /api/ → :8081,
// so no CORS). The Auth0 SDK mints/refreshes the access token via its
// /auth/access-token endpoint.
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getAccessToken();
  const res = await fetch(path, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // opaque failure; keep the status message
    }
    throw new Error(message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
