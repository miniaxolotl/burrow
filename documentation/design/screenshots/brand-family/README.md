# Burrow brand family refresh

Captured 20 October 2026 from the production build at `http://10.8.0.5:4182`.
Desktop: Chromium, 1440 × 1000. Mobile: WebKit, 390 × 1000 with touch. Captures
use reduced motion and wait for all images and fonts. These are browser captures.

- [Desktop](home-desktop.png) and [mobile](home-mobile.png): approved 07L / Arc
  owl, IBM Plex Sans/Mono, clay accents and terminal-led workflows.
- [Design kit](design-kit.png): approved artwork, native icons, licensed fonts,
  palette, social image and campaign banner.
- [Route review](route-review.json): expected responses, no page exceptions,
  document overflow or WCAG A/AA violations in both browser engines.

Website lint, types and production build pass. All three Go modules pass vet and
tests using Go 1.26.2. Browser review checks tunnel selection, request logs,
returning to the selected tunnel and denied-clipboard recovery. Earlier homepage
review also covers 320px, 390px and 1440px without overflow or contrast findings.

The original Arc geometry remains in `src/lib/brand-artwork.json` and the shared
mark/logo components. Imported collection 04 files retain their source-byte
checksums in `packages/website/design-kit/asset-manifest.json`. Terminal traffic
and URLs are labelled examples; website identity remains `useburrow.app`.
