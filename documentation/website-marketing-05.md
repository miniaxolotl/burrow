# Website product presentation

The website leads with giving a local app a public HTTPS link. The main action is
installation. Short examples explain preview sharing, webhook testing and running
the open-source server on your own infrastructure.

The service workflow connects a Frontend on port 3000 and API on port 8080 in one
HTML example. Selecting a service updates its public-link example, matching local
port, newest-first request metadata and owner-scoped follow command. Status and
duration are separate columns; a prepared 404/202 API example explains missing
routes and acknowledgement versus completed processing. Both panels stay mounted
in an intrinsic grid track to preserve the surrounding layout. The examples make
no requests to the reserved `example.com` hosts and open no tunnels. IBM Plex Sans handles
labels and explanations; Plex Mono is reserved for commands, URLs and paths.
Native service buttons retain keyboard focus, identify their controlled panels
and announce the selected example without making the full table a live region.
Terminal scenario labels and request routes
remain readable within the existing stable demonstration. The approved Arc logo,
IBM Plex type and Apple touch icon remain unchanged. Adaptive SVG and explicit
light/dark ICO/PNG fallbacks follow the browser colour scheme independently of
the light page; imported bytes are traced in
`packages/website/public/marketing-kit.json`.

**No new marketing-kit website graphics are shipped for Burrow, including share
compositions.** Existing terminal-based social artwork remains. The identity-only
manifest and `tests/marketing-kit.spec.ts` enforce this boundary and check served
favicon hashes and theme links. Existing installation, manager, request inspection
and clipboard behavior remain part of the browser regression suite.

The family polish retains Plex, the Arc identity, clay actions and the existing
terminal layout. The headline uses full ink; legal text is 16px with readable
metadata, and support/contact use the same restrained heading scale as the
other family sites. Public-only neutral paper/white surfaces and focus treatments
do not restyle the CLI or change the tunnel service.

The interactive terminal now includes **Local app offline**, showing a connected
tunnel with a sample 502 response and recovery guidance. Its labels make clear
that traffic is fictional, no tunnel is created and the browser never probes
localhost. Existing scenario, installation-command and manager controls remain
playable. A tunnel is not a deployment, and HTTPS does not authenticate the
application. See the [family-polish review](design/screenshots/marketing-family-polish/README.md).

Research direction: prove webhook and local-preview workflows, not an exclusive
“no-account” claim. [Cloudflare Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/)
also offers no-account setup; [ngrok](https://ngrok.com/docs/pricing-limits/free-plan-limits)
and [Tailscale Funnel](https://tailscale.com/kb/1223/funnel) illustrate different
setup/access trade-offs. Acquisition experiments remain recommendations, not
launched campaigns or validated market demand.
The [earlier four-width review](design/screenshots/marketing-polish-20261106/README.md)
is retained as dated evidence, not the current service-selector presentation.

Current help/contact routes and designated public email are documented in
[Website support and contact](website-support-contact.md). Earlier verification
counts below remain historical, not the current regression-suite totals.

## Copy follow-up

The Collection 05 sections below are historical; their test counts and artwork
descriptions do not describe the current source or a new production acceptance.

The 21 October follow-up shortens workflow headings, FAQs and footer referrals,
uses a plain missing-page message and fixes repeated legal-page titles. Public
pages no longer promote brand materials or downloads. The install action,
example labels and approved 07L Arc artwork remain.

Website formatting/lint/types/build, 11 browser cases, three Go module test
suites, 11 release contracts and workflow lint pass. Public-page reviews cover
320/390/768/1440px, legal and missing pages, image loading, accessibility and
absence of brand-download links.

## Assets

- Workflow illustrations come from Collection 05 in the local marketing dataset.
- The approved 07L Arc identity and IBM Plex Sans/Mono typography remain.
- The share image, brand page, product description and media kit are refreshed.
- `packages/website/public/materials/manifest.json` records sources and hashes.
- The media kit includes the outlined identities, fonts, licenses and artwork.

The exporter is `marketing/_source/export-websites-05.mjs`, using `artwork.mjs`,
the approved identity data and design guidelines. Graphics are labeled as examples.
Installation commands and server documentation links use the shipped product.

## Review

Copy uses standard punctuation and direct calls to install or try a workflow.
Browser review covers phone and desktop layouts, asset downloads, canonical
links, accessibility and the terminal previews. Existing telemetry tests exercise
installation outcomes, clipboard failures, masking and opt-out behavior.

Verified 21 October 2026: website lint/types/build, 11 browser tests, all three
Go module test suites, 11 release contracts and workflow lint pass. Additional
reviews at 320, 390, 768 and 1440px pass accessibility, overflow, image and
download checks. Identity hashes, font licenses and media-kit contents verify.

Publication follows `deploy.md` through protected development, Release Please and
the selected frozen production candidate. The deployment verifies the real tunnel.
