# Burrow website

The `website` workspace package is Burrow’s marketing site. It uses Next.js App Router, React, TypeScript, and Tailwind CSS v4, with locally served fonts and assets.

## Bramble Labs design family

Burrow and Fern share a compact editorial template: matching typography, spacing, navigation, button shapes, feature rows, FAQs, and Bramble Labs attribution. Burrow uses clay and charcoal; Fern uses botanical green and sage. Product previews are built from React, CSS, and SVG rather than app screenshots. Both websites remain independent workspace packages in their own repositories.

## Development

From the repository root (Node.js 22+, pnpm 10):

```bash
pnpm install --frozen-lockfile
pnpm dev
```

Open <http://localhost:3000>. To use another port:

```bash
pnpm --filter website dev --port 3001
```

## Production

Copy `packages/website/.env.example` to `packages/website/.env.local` and set `SITE_URL` to the website’s public origin **before building**. This is the marketing website URL, not necessarily the `burrowd` server address. Vercel’s production URL and Railway’s public domain are also recognized when `SITE_URL` is absent. Localhost is the development fallback and is marked `noindex`.

```bash
pnpm build:website
pnpm start:website
```

Deploy as a Node.js application from the monorepo root, using the commands above. `PORT` is supported by `next start`. The landing page is rendered at request time with `connection()`; this is not a static export. No Redis, database, or tunnel server is needed to run the website.

### Vercel

Import this repository with the **Next.js** preset and root directory **`packages/website`**. Keep workspace files outside the root directory available, use `pnpm build` as the build command, and set `SITE_URL` to the final website origin before deploying. The standard Next.js output settings apply.

## Checks

```bash
pnpm lint:website
pnpm typecheck:website
pnpm build:website
```

The package extends `@lib/typescript-config`. It uses Next.js’s recommended accessibility, React, TypeScript, and Core Web Vitals lint rules with ESLint 9, the version supported by those plugins.

## Content and sharing

- `src/app/page.tsx`: server-rendered landing page and feature copy.
- `src/components/`: small interactive previews, clipboard controls, navigation, and native sharing with a manual-copy fallback.
- `src/content/faq.ts`: FAQ copy shared by the page and structured data.
- `src/lib/site.ts`: public origin, product description, and repository/package links.
- `src/app/globals.css`: Tailwind theme, responsive layouts, and reduced-motion support.
- `src/app/opengraph-image.tsx`: branded 1200 × 630 social card, also served as the Twitter image.
- `src/app/robots.ts`, `sitemap.ts`: crawler metadata.
- `src/components/manager-preview.tsx`: responsive, code-built tunnel manager and request-log views with selectable example tunnels and copyable CLI commands.

Page content, installation commands, documentation links, and FAQ work without JavaScript. The terminal illustration uses labeled example traffic; it does not open a real tunnel. Clipboard, preview selectors, and native sharing are progressively enhanced client components. Fonts and social-image generation do not require a third-party font service.
