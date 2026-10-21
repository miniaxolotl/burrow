# Collection 05 website materials

The website leads with giving a local app a public HTTPS link. The main action is
installation. Short examples explain preview sharing, webhook testing and running
the open-source server on your own infrastructure.

## Copy follow-up

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
