# Website consent delivery

Verified on **22 October 2026 UTC** at <https://useburrow.app>.

- Source: [v0.3.11](https://github.com/miniaxolotl/burrow/releases/tag/v0.3.11), `2741fdf60ca8ac0dc9fa8de69de563b7c6df0e62`.
- Frozen production [#54](https://github.com/miniaxolotl/burrow/pull/54): `c860fddc35090fdb7d93074bfc75368bff0345c0`; source and production trees match.
- Production [CI 39723057650](https://github.com/miniaxolotl/burrow/actions/runs/39723057650) and [Deploy 39723057655](https://github.com/miniaxolotl/burrow/actions/runs/39723057655) passed.
- Vercel deployment `dpl_69vaY5HFfBwwV7oht1vJr38haEt6` is READY and the public alias serves that production SHA.

## Behavior

Pinned Vanilla CookieConsent **3.1.0** stores separate Analytics and Advertising
choices for 180 days. A compact branded notice offers **Accept all**, **No thanks**
and **Settings**; the footer provides **Cookie settings**. The approved Burrow
07L / Arc design is retained. See the [approved screenshots](design/screenshots/cookie-consent/README.md).
The live Analytics description reads **“Understand site usage and performance.”**

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
The coordinated v0.3.11 release also passes the real HTTPS tunnel-forwarding check.
All five downloaded npm artifacts match their registry integrity, version and
repository, advertise provenance, and the Linux x64 CLI starts at 0.3.11. The
published server image is
`ghcr.io/miniaxolotl/burrowd@sha256:07753ebc32e2a23d253e6bbefbecae0e6b1391b279834e8ee0a502cfc69d72d1`.
The earlier v0.3.10 registry interruption recovered on deployment attempt4 before
this patch was promoted. The patch promotion's OpenTofu download timeout recovered
on a same-candidate check retry through normal branch protection.
No paid campaign was configured.
