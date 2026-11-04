# Burrow terminal-led landing page

Reviewed local optimized-build captures of the live-layout-led polish pass.
[Capture provenance](capture.json) records the source fingerprint, build ID,
browser, viewports and synthetic-data boundary.

## Refinement review

Phone guides use a native disclosure instead of placing the full resource list
before the article; useful content starts 120px sooner at 390px. Desktop keeps its
sidebar, and all guide links remain available without JavaScript.

| Guide navigation expanded                                             | Manual-copy recovery                                                  |
| --------------------------------------------------------------------- | --------------------------------------------------------------------- |
| ![Native guide disclosure on a phone](docs-client-navigation-390.png) | ![Native share fallback with a selected link](share-fallback-390.png) |

Recovery uses synthetic denied browser sharing/clipboard access. Done/Escape
returns focus and the footer/page/scroll do not move (previously 74px at 390px).
All 41 browser cases pass. Four-width review reports zero inspector failures,
automated accessibility violations, overflow or demo-height changes.

Captured from the local optimized Next.js build of `feature/website-product-refresh`.
Rust actions, IBM Plex type and working dark terminal examples distinguish the tool.
Installation instructions stay on paper with a matching readable foreground; manager
tabs are readable outside the terminal in their selected and unselected states.

| Desktop — 1440 × 900                  | Phone — 390 × 900                  |
| ------------------------------------- | ---------------------------------- |
| ![Burrow landing page](home-1440.png) | ![Burrow on a phone](home-390.png) |

[Tablet, 768 × 900](home-768.png) · [Small phone, 320 × 900](home-320.png) · [Full desktop page](home-page-1440.png) ·
[Full phone page](home-page-390.png)

| Compact footer — desktop                                           | Compact footer — phone                            |
| ------------------------------------------------------------------ | ------------------------------------------------- |
| ![Approved Arc signature and compact footer](home-footer-1440.png) | ![Compact footer on a phone](home-footer-390.png) |

The 4 November kit guidelines govern Plex Sans hero/section headings, Plex Mono
commands and the approved Arc signature in the compact footer. There is no
separate giant-name row; footer controls retain a minimum 44px target.

[Installation, desktop](walkthrough-1440.png) · [Phone](walkthrough-390.png) ·
[Selected webhook, desktop](webhook-1440.png) · [Phone](webhook-390.png) ·
[Tablet page](home-page-768.png) · [Small-phone page](home-page-320.png) ·
[Keyboard focus](home-focus-1440.png) · [200% CSS zoom reflow](home-zoom-200.png).
The zoom capture checks CSS reflow, not native browser zoom or physical-device accessibility.

Commands use the implemented CLI. The terminal and manager contain synthetic URLs,
ports and requests, never a live tunnel or customer traffic. The page distinguishes
public app access from owner-only management, and states platform and telemetry limits.
The code photograph is a local derivative of a retained marketing-kit master;
its source, licence and hashes are in the
[asset manifest](../../../../packages/website/public/images/kit/manifest.json).
[Client setup and troubleshooting](docs-client-page-390.png) use the maintained guide.

Optional cookies were declined through the real notice before capture. Consent,
clipboard failure and keyboard interactions are independently covered by browser
tests. These captures are local source evidence, not a deployment or live tunnel test.
