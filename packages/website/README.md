# Burrow website

The `website` workspace package is Burrow’s marketing site. It uses Next.js App Router, React, TypeScript, and Tailwind CSS v4, with locally served fonts and assets.

## Bramble Labs Inc. design family

The family shares warm paper, an 8px spacing rhythm, clear controls and restrained motion. Burrow has its own IBM Plex Sans and IBM Plex Mono typography, clay palette and terminal-led story. [The design kit](design-kit/README.md) contains tokens, approved assets, licences and provenance. Open `/brand-kit/index.html` for the visual guide and downloads. All three products link to https://bramblelabs.co.

## Selected identity: 07L / Arc

The website uses the selected burrowing-owl mark with a curved outward passage and the approved, outlined Burrow wordmark. `src/lib/brand-artwork.json` preserves the exact selected paths and lockup spacing. The header, footer, 404 page, Apple icon and social cards share these paths through `BurrowLogo` and `BurrowMark`.

Downloadable original SVGs are in `public/brand/`. Collection 04 assets are in `public/brand-kit/`, including native icons and approved outlined social artwork. `src/app/icon.svg` and `favicon.ico` use the supplied optical exports. The complete selected pack is saved in the sibling marketing studio at `marketing/burrow/identity/`. The website build has no dependency on that folder.

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
