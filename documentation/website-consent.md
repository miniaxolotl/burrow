# Website consent delivery

Verified on **22 October 2026 UTC** at <https://useburrow.app>.

- Source: [v0.3.10](https://github.com/miniaxolotl/burrow/releases/tag/v0.3.10), `662b3837a1025b34d7e77a2781ceecd8b2a85558`.
- Frozen production [#50](https://github.com/miniaxolotl/burrow/pull/50): `ee1bb60936f96b9fd85dcaed4b4222eae7fa2398`; source and production trees match.
- Production [CI 39710513609](https://github.com/miniaxolotl/burrow/actions/runs/39710513609) passed.
- Vercel deployment `dpl_3KQBjNyHLJqbnPtgDempU5VHCrXm` is READY on that production SHA.

## Behavior

Pinned Vanilla CookieConsent **3.1.0** stores separate Analytics and Advertising
choices for 180 days. A compact branded notice offers **Accept all**, **No thanks**
and **Settings**; the footer provides **Cookie settings**. The approved Burrow
07L / Arc design is retained. See the [approved screenshots](design/screenshots/cookie-consent/README.md).

Optional browser analytics, diagnostics and Vercel measurements require Analytics
consent. Advertising consent independently enables the production public-route
tag loader. DNT, GPC and unavailable storage prevent optional collection.
Revocation reloads with tracking disabled and unloads advertising.

## Provider mapping

The Bramble Labs Inc. GTM account is `6399006064`. Published container
**GTM-PGLVNMZ8**, version **2**, maps only to advertiser
`KHM6GIJGMRA6LP6CITXR4S` and pixel `VU4V4ZQWMFAP7LJEEZGESJ`.
The DOM Ready trigger admits only `useburrow.app` public marketing paths
`/`, `/privacy` and `/terms`. No email identity, custom data or conversion value
is configured. AdRoll reads back **Use your existing Consent Banner**.

## Production acceptance

Fresh visits and remembered dismissal generated no optional provider requests.
Advertising-only consent loaded the matching GTM container and pixel with an
HTTP 200 pixel response, while analytics stayed off. Revocation unloaded the tag
and remained effective after reload. Desktop and 390px mobile layout, focus and
browser-error checks passed. AdRoll subsequently reported **Pixel Active**.

The implementation passed formatting, lint, types, production build, 20 browser
scenarios and applicable release/workflow gates before protected promotion.
The website is verified live. Coordinated package/server release
[39710513610](https://github.com/miniaxolotl/burrow/actions/runs/39710513610) separately
encountered the npm registry incident at credential preflight and awaits recovery.
No paid campaign was configured.
