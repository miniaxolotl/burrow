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
