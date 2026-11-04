# Burrow website

The `website` workspace package is Burrow’s marketing site. It uses Next.js App Router, React, TypeScript, and Tailwind CSS v4, with locally served fonts and assets.

## Bramble Labs Inc. design family

The family shares warm paper, an 8px spacing rhythm, clear controls and restrained motion. Burrow has its own IBM Plex Sans and IBM Plex Mono typography, clay palette and terminal-led story. [The design kit](design-kit/README.md) contains tokens, approved assets, licences and provenance. Open `/brand-kit/index.html` for the visual guide and downloads. All three products link to https://bramblelabs.co.

## Selected identity: 07L / Arc

The homepage request diagram follows the selected browser, webhook or API
scenario through Burrow's public HTTPS endpoint to the matching localhost port.
It is a labelled HTML sequence that stacks vertically on phones. Example traffic
never opens a tunnel or contacts an example hostname. Quick start pairs copyable
commands with prerequisites, local-port instructions and a browser verification
step; copying alone is not presented as a successful installation.

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

`/docs` builds public client/server references and worked examples from the
committed `documentation/` directory. The manifest drives navigation and sitemap
coverage; retain monorepo files outside the website root during the build. See
[reference maintenance and discovery](../../documentation/website-discovery.md).

- `src/app/page.tsx`: server-rendered landing page and feature copy.
- `src/components/`: small interactive previews, clipboard controls, navigation, and native sharing with a manual-copy fallback.
- `src/content/faq.ts`: FAQ copy shared by the page and structured data.
- `src/lib/site.ts`: public origin, product description, and repository/package links.
- `src/app/globals.css`: Tailwind theme, responsive layouts, and reduced-motion support.
- `src/app/opengraph-image.tsx`: approved 1200 × 628 social PNG, also served as the Twitter image.
- `src/app/robots.ts`, `sitemap.ts`: crawler metadata.
- `src/components/manager-preview.tsx`: responsive, code-built tunnel manager and request-log views with selectable example tunnels and copyable CLI commands.

Page content, installation commands, documentation links, and FAQ work without JavaScript. The terminal illustration uses labeled example traffic; it does not open a real tunnel. Clipboard, preview selectors, and native sharing are progressively enhanced client components. Fonts and social-image generation do not require a third-party font service.

# Product presentation

The marketing page leads with a concrete HTTPS tunnel example and working CLI
install/copy controls. It explains public URLs, multi-port management, request
logs, reconnection and automatic private ownership. It distinguishes owner-only
management from the public app behind a tunnel: users must add app authentication
before exposing sensitive data. macOS/Linux x64/ARM64, npm's Node requirement,
native binaries, self-hosting and telemetry opt-out are explicit.

The four Bramble sites share plain navigation, single-face headings, readable copy,
filled product demonstrations and consistent spacing. Existing marks, local fonts
and colours remain. “Your local app. A public link.” sits beside a working
terminal on paper, refining the live split layout. Compact rust actions and IBM
Plex typography retain the identity. The installation
guide follows the opening; there is only one installation-command source.
Preview, webhook and API selectors update the local port, request,
status and explanatory copy together, without opening a tunnel or requesting an
example hostname. The compact footer uses the approved outlined signature in its
brand column rather than a separate giant-name row. Shared chrome uses
36px header / 38px primary actions for fine pointers, 44px coarse-pointer targets,
an 80px desktop / 72px phone header and a 1280px content frame.
The manager is an interactive sample,
not a live tunnel.
The owning marketing-kit guide is `marketing/guidelines/burrow.md`, generated
from `marketing/_source/website-guidelines.mjs`. Its 4 November website rules
specify Plex Sans 500 headings, Plex Mono for real commands,
optical tracking and two-line headings. Footer top padding is 40px desktop /
32px phone; the approved signature stays proportional at up to 280px / 220px.
The website owns its CSS/assets, with no build-time dependency on the sibling kit.
Research drew on [Linear](https://linear.app), [Craft](https://www.craft.do),
[Things](https://culturedcode.com/things/), [Retro](https://retro.app),
[Raycast](https://www.raycast.com) and [Panic](https://panic.com), without importing
their assets or promises. The decorative connection illustration is replaced by
the command/output itself.

The client guide at `/docs/client` owns prerequisites, package/binary installation,
local/public verification, ownership and troubleshooting. `/docs/workflows` owns
the runnable webhook example and cleanup. The landing quick start links to those
guides rather than duplicating the full reference. Public prose is sourced from
the explicit documentation allowlist, not arbitrary repository files.

The supporting code photograph is a local derivative of the marketing kit's
master, not a Burrow screenshot. `public/images/kit/manifest.json` retains source
and output hashes, dimensions and licensing; the website needs no sibling kit at
runtime.

Scenario and manager panels remain mounted in overlapping intrinsic grid tracks.
Selection changes the visible example without remounting the terminal or moving
surrounding sections. Inactive content is inert and hidden from assistive
technology; only the explicit replay action restarts the example. The stability
tests cover scenarios, tunnel selection and request views at four widths.

Reviewed captures live in [the product-redesign gallery](../../documentation/design/screenshots/product-redesign/README.md).
Source verification is not a hosted tunnel test or deployment. Existing protected
release and website checks remain required.

`/llms.txt` supplements initial HTML with public CLI/product facts and documentation
links; an alternate text link advertises it in the head. This does not promise
AI ingestion or citation. Preview/development and `SITE_NOINDEX=true` serve noindex,
disallow-all robots rules and an empty sitemap. Production metadata and sitemap
origins come from `SITE_URL`; robots directives are not access control.
