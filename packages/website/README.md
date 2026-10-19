# Burrow website

The `website` workspace package is Burrow’s marketing site. It uses Next.js App Router, React, TypeScript, and Tailwind CSS v4, with locally served fonts and assets.

## Bramble Labs Inc. design family

Burrow and Grove share a developer-focused design: warm ivory surfaces, DM Sans with Instrument Serif accents, monospace details, terminal previews, and direct installation paths. Clay accents distinguish Burrow from Grove’s green. Fern shares the visual foundations with a photography-led consumer experience. All three link through a common product-family footer to the company at https://bramblelabs.co.

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

The primary website is <https://useburrow.app>. Use the **Next.js** preset and root directory **`packages/website`**. Keep workspace files outside the root directory available, use `pnpm build` as the build command, and set `SITE_URL=https://useburrow.app`. The standard Next.js output settings apply.

Vercel Web Analytics is integrated through `@vercel/analytics/next` in the root layout and enabled in the Vercel project. Reports are available in the project's Analytics dashboard. The footer links to website privacy information at https://bramblelabs.co/privacy.

Vercel Speed Insights uses `@vercel/speed-insights/next` through
`WebsiteSpeedInsights` only when `VERCEL_ENV=production`. It measures loading,
responsiveness, and visual stability on `/`, `/privacy`, and `/terms`. The
`beforeSend` filter removes query strings and fragments and drops all other paths;
it does not instrument tunnels. Local and preview builds do not collect
performance metrics. View results in the project's **Speed Insights** dashboard.
The SDK uses Vercel's injected intake configuration without an extra key. Update
`publicPaths` in the component when adding public pages.

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
