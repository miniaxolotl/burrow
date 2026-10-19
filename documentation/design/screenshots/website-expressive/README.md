# Burrow website polish

Captured October 19, 2026 with Playwright Chromium at `127.0.0.1:4191`.
Desktop is 1440px; mobile is 390px, with reduced motion for stable screenshots.

- [Desktop homepage](home-desktop.png) and [mobile homepage](home-mobile.png):
  terracotta tunnels, a terminal-led hero and three developer workflow stories.
- [Privacy page](privacy-desktop.png): shared legal layout, Burrow-specific
  traffic/credential/retention information and confirmed support contact.

## Verification

- `pnpm lint:website`, `pnpm typecheck:website`, `pnpm build:website` pass.
- [Route review](route-review.json): homepage, privacy, terms and 404 at
  320/390/768/1024/1440px; expected HTTP responses, no browser exceptions,
  no horizontal page overflow, and no axe WCAG A/AA violations including contrast.
- Chromium interaction review passes: mobile menu Escape/focus/outside/resize
  dismissal, legal-to-home section navigation, npm/pnpm installation switching,
  copied command content and clipboard-denied recovery.

Illustrations are authored SVG/CSS and illustrative terminal examples.

## Production publication — October 19, 2026

- Live: <https://useburrow.app>
- Website source: `fab5901f74516e08616ec3380f54c8fdde1a0cbf`;
  [PR #17](https://github.com/miniaxolotl/burrow/pull/17).
- Vercel deployment: `dpl_6kzZcFCE6xZuiYZigL6m2rPSVYAT`, independently read
  back as READY with that source SHA and the production domain alias.
- [CI run 39222173528](https://github.com/miniaxolotl/burrow/actions/runs/39222173528)
  passed website, build, infrastructure, lint, and test checks.
- [Live route review](live-route-review.json) covers all four public routes
  at five widths with expected responses, no page errors, document overflow,
  or automated WCAG A/AA violations.
- [Live journeys](live-journeys.json) pass package-manager switching, command
  copying, navigation, metadata and legal-link checks.
