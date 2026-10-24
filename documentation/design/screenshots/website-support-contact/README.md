# Support/contact website evidence

Captured 24 October 2026 from `feat/website-support-contact`, based on `af8fed2`,
with the contact-migration diff present. Local production-build Chromium evidence,
not a production deployment. No customer data or external fixture media is used.

| Capture                              | Scenario                                                                                     |
| ------------------------------------ | -------------------------------------------------------------------------------------------- |
| [Policy contact](policy-contact.png) | `/privacy`, 1440 × 1000 viewport; clipped to `.legal-contact` to show the designated mailbox |

Build the website with documented synthetic PostHog/Sentry settings, start Next
on a free localhost port, wait for `document.fonts.ready`, and capture the stated
route/element with Playwright. Reviewed for readable text and complete wrapping.
