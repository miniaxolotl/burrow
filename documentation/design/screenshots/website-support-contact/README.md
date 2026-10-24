# Support/contact website evidence

Captured 24 October 2026 from `feat/website-support-contact`, based on `af8fed2`,
with the contact-migration diff present. Local production-build Chromium evidence,
not a production deployment. No customer data or external fixture media is used.

| Capture                              | Scenario                                                                                     |
| ------------------------------------ | -------------------------------------------------------------------------------------------- |
| [Policy contact](policy-contact.png) | `/privacy`, 1440 × 1000 viewport; clipped to `.legal-contact` to show the designated mailbox |
| [Support desktop](support-1440.png)  | `/support`, 1440px                                                                           |
| [Support phone](support-390.png)     | `/support`, 390px                                                                            |
| [Contact desktop](contact-1440.png)  | `/contact`, 1440px                                                                           |
| [Contact phone](contact-390.png)     | `/contact`, 390px                                                                            |

Page captures use contact-migration commit `c4a779b` plus the page diff, a
1000px-tall viewport, reduced motion, and JavaScript disabled. The eight separate
axe audits run with JavaScript enabled and an explicit browser context, as axe
requires. Capture review covers readable hierarchy, wrapping and footer separation.

Build the website with documented synthetic PostHog/Sentry settings, start Next
on a free localhost port, wait for `document.fonts.ready`, and capture the stated
route/element with Playwright. Reviewed for readable text and complete wrapping.
