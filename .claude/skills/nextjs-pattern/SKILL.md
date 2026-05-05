---
name: nextjs-pattern
description: Use this skill whenever the user asks to build, modify, or review Next.js code in apps/web or apps/dashboard. Covers App Router patterns, Server Components, shadcn/ui, Tailwind, Zod validation, TanStack Query, server actions, and DevOps Access-specific data-fetching conventions. Triggers on "Next.js", "React component", "page.tsx", "Server Component", "use client", "shadcn".
---

# DevOps Access Next.js Conventions

## Stack

- Next.js 14+ App Router (NOT Pages Router)
- React 18 with Server Components as default
- Tailwind CSS
- shadcn/ui components (copy-pasted into `components/ui/`, NOT npm-installed)
- Zod for all external input validation
- TanStack Query for client-side data caching
- Auth0 for authentication (server-side verification via JWKS)

## Directory layout
apps/<web|dashboard>/
├── src/
│   ├── app/
│   │   ├── layout.tsx
│   │   ├── page.tsx
│   │   ├── (marketing)/         # route group
│   │   ├── (app)/               # authenticated routes
│   │   ├── api/                 # API routes (use sparingly)
│   │   └── globals.css
│   ├── components/
│   │   ├── ui/                  # shadcn components
│   │   └── feature/             # feature-specific components
│   ├── lib/
│   │   ├── auth.ts              # Auth0 verification
│   │   ├── api.ts               # server-side API client
│   │   └── utils.ts
│   ├── hooks/                   # client-side React hooks
│   └── types/
├── public/
├── next.config.js
├── tailwind.config.ts
└── tsconfig.json
## Server Component default

```tsx
// app/(app)/dashboard/page.tsx — Server Component by default
import { getTenants } from '@/lib/api';
import { TenantList } from '@/components/feature/tenant-list';

export default async function DashboardPage() {
  const tenants = await getTenants();  // runs on server, never ships to client
  return <TenantList initial={tenants} />;
}
```

## Client Component when needed

```tsx
// components/feature/tenant-list.tsx
'use client';

import { useQuery } from '@tanstack/react-query';
import { fetchTenants } from '@/lib/api-client';
import type { Tenant } from '@/types';

export function TenantList({ initial }: { initial: Tenant[] }) {
  const { data } = useQuery({
    queryKey: ['tenants'],
    queryFn: fetchTenants,
    initialData: initial,
    staleTime: 30_000,
  });

  return <ul>{data?.map(t => <li key={t.id}>{t.name}</li>)}</ul>;
}
```

## Server action with Zod

```tsx
// app/(app)/settings/actions.ts
'use server';

import { z } from 'zod';
import { revalidatePath } from 'next/cache';

const updateProfileSchema = z.object({
  name: z.string().min(1).max(100),
  email: z.string().email(),
});

export async function updateProfile(formData: FormData) {
  const parsed = updateProfileSchema.safeParse({
    name: formData.get('name'),
    email: formData.get('email'),
  });

  if (!parsed.success) {
    return { error: parsed.error.flatten() };
  }

  // ... mutation
  revalidatePath('/settings');
  return { success: true };
}
```

## Hard rules

1. Every route has a `loading.tsx` and `error.tsx` at its level.
2. Server Components fetch data directly; never use `useEffect` + fetch on the client for initial data.
3. All external inputs (forms, URL params, API responses) pass through Zod before use.
4. Images always via `next/image` with explicit `width`/`height` or `fill`.
5. No client-side `localStorage`/`sessionStorage` for auth tokens. Use httpOnly cookies only.
6. Server actions for mutations; API routes only when a third-party webhook needs them.
7. Tailwind utility classes on elements; never inline styles. Use shadcn's `cn()` helper for conditional classes.
8. Every form uses `useFormState` + `useFormStatus` for progressive enhancement.

## Common pitfalls

- `'use client'` is viral: any Server Component that imports a Client Component chain turns everything downstream client-side. Be surgical.
- `fetch()` in Server Components auto-caches. Use `{ cache: 'no-store' }` or `{ next: { revalidate: 0 } }` for fresh data.
- Hydration mismatches: never use `Date.now()` or `Math.random()` in a component that renders server+client without `suppressHydrationWarning`.
- Tailwind purges unused classes at build time. Don't construct class names dynamically (e.g., `bg-${color}-500`) — they won't ship. Use a lookup map instead.
- Don't import server code (DB, secrets) from Client Components. Use the `server-only` package to enforce at build time.
