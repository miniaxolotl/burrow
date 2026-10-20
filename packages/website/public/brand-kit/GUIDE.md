# Burrow website design kit

Collection 04, 20 October 2026. Canonical home: https://useburrow.app.

## Files

Open `/brand-kit/index.html` for the visual specimen and individual downloads.
`tokens.css` supplies runtime colours; `tokens.json` records layout and motion.
`asset-manifest.json` records imported source paths and SHA-256 checksums.
`../public/brand-kit/` contains primary/inverse logos, outlined wordmarks, symbols,
native-size icons, licensed local WOFF2 fonts, social art and a campaign banner.
The PDF is the printable brand guide. Source: marketing collection 04.

## Identity

The approved **07L / Arc** owl and outlined wordmark are fixed. Preserve their
paths, proportions and optical padding. The existing `BurrowLogo` and `BurrowMark`
components render that approved artwork. Use native-size favicons for tiny marks.
The website identity is **useburrow.app**. The hosted tunnel service is a separate
domain; demonstration URLs and requests must be labelled as examples.

Use **IBM Plex Sans**: 600 headlines, 500 subheads and controls, 400 reading.
Use **IBM Plex Mono**: 400 code and commands, 500 technical labels. Keep headlines
upright and command letter spacing natural. Retain the font licences.
Paper `#f7f6f0`, ink `#253e31`, clay `#9d4930`, support `#efbc96`.

## Layout and motion

The family uses a 1280px frame, 20/32/48px gutters and 8px spacing. Main controls
are 48px tall, other targets at least 44px. Use 12px buttons and 16 to 24px panels.
Keep the terminal precise and readable. One outcome, one command, one next action.
`src/app/brand-family.css` applies the collection after structural styles.

Entrances take 700 to 900ms, one-time reveals 600ms and feedback 200ms. All content
is visible without JavaScript; reduced motion uses a static scene. Keep clipboard
fallbacks, keyboard tunnel selection and accessible request inspection working.

## Voice and product truth

Localhost, shared. Name the task, show the command, explain the result. Speak to
developers sharing demos, receiving webhooks and inspecting requests. Use ordinary
punctuation. Install: `npm i -g @miniaxolotl/burrowctl`. Start:
`burrowctl tunnel 3000`. Current native support is macOS/Linux, x64/ARM64.
Avoid unsupported Windows, deployment or private-access promises.

Use https://useburrow.app, https://bramblelabs.co, https://usefern.app and
https://usegrove.cloud for the product family. Preserve the existing WireGuard
preview access when running local reviews.

## Verification

Run website lint, types and production build plus repository Go checks. Review
desktop and 320/390px layouts, commands, manual-copy recovery, request inspection,
navigation, reduced motion, icons, social images and accessibility. Store browser
captures and their provenance under documentation/design/screenshots.
