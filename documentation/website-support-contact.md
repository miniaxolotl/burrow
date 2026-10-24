# Website support and contact

## Public help pages

`/support` covers local-app checks, installation/documentation, connection and
webhook troubleshooting, and the public nature of a tunnel. `/contact` covers
feedback, safe problem reports, and private security/privacy inquiries. Both use
the designated mailbox, not a web form backend, and work without JavaScript.
Footer and missing-page help links now reach these pages. Each page has its own
canonical/social metadata; the sitemap retains the existing preview/local noindex guard.

Current website source uses **support@useburrow.app** for general, support,
privacy, and private security inquiries. The public-only contact module at
`packages/website/src/lib/contact.ts` is shared by policies, footer, and recovery.
Keep unrelated author attribution and dated release evidence unchanged.

`packages/website/tests/support-contact.spec.ts` guards the public mailbox and
absence of obsolete current website contact links. Browser checks use a real
production build. No CLI, protocol, tunnel server, provider, or deployment
behavior is changed by this contact migration. Never email credentials or
private tunnel traffic as part of a support report.

The contact regression first failed on the obsolete links; after migration all
**24 Chromium cases**, root/site lint, typecheck, build, Go vet and Go tests pass.
The optional live tunnel target was not configured; no live tunnel acceptance is
claimed. [Reviewed contact capture](design/screenshots/website-support-contact/README.md).

New page tests first failed on 404 responses. The final full **27-case Chromium
suite**, root/site lint, website types/build and Go vet/tests pass. A second build
with `SITE_URL=https://useburrow.app` passes all four support/contact tests,
including public sitemap inclusion and canonical URLs. Eight local axe WCAG A/AA
audits pass (two pages at 320/390/768/1440px), using axe 4.14.0 from the workspace's
Grove development tooling; no new Burrow dependency was added. Captures are
JavaScript-disabled browser evidence, not native-device or live-deployment evidence.
