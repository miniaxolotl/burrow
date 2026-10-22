# Product observability

## Google Search Console

Verified on **21 October 2026 UTC**: the
[useburrow.app Domain property](https://search.google.com/search-console?resource_id=sc-domain%3Auseburrow.app)
reports **You are a verified owner**. The apex Cloudflare TXT record was read
back through its API and public Google DNS. Keep that verification record in DNS
to retain ownership; the account, exact record and private evidence are recorded
in the original checkout's ignored `secrets.md`.

Google processed [the production sitemap](https://useburrow.app/sitemap.xml)
successfully on the same date: **Success**, **3 discovered pages**, zero videos.
Both `/robots.txt` and `/sitemap.xml` return HTTP 200. Discovered pages are not an
indexed-page count; initial indexing and search-performance reports are collecting
data. Use **Sitemaps** for processing errors, **Pages** for indexing, and
**Performance** for search queries, impressions, clicks and position. Search
Console complements PostHog acquisition and behavior reporting through Google's
search data and DNS ownership verification.

## Website implementation

The website integrates PostHog US project **677269** and Sentry project
**burrow-website** in the existing `bramble-labs-inc` team. CLI/server integration
shares that PostHog project, with separate Sentry `burrow-cli`/`burrow-server`
projects. Release **0.3.7** is published and live; delivery evidence is below.

- [Acquisition and activation](https://us.posthog.com/project/677269/dashboard/2236993)
- [Experience and reliability](https://us.posthog.com/project/677269/dashboard/2236994)
- Sentry project ID: `4512290792275968`.

### Collection

Browser identities are random, stored locally, and never attached to tunnel
credentials, hostnames, ports, names or email. PostHog person profiles are disabled.
The final event allowlist covers public-route page views/leaves, CTA/product/docs
links, successful **installation** command copies, clipboard operation outcomes,
numeric web vitals, actual experiment exposure and numeric survey ratings. Copying
a demonstration command is not an installation conversion. Queries, fragments,
unknown paths, arbitrary properties and attribution objects are removed.

Referral links use `utm_source=burrow&utm_medium=referral&utm_campaign=product_family`.
Project dashboards use event-level `$referring_domain`. Cross-product/browser/CLI
identity correlation is not inferred from these referrals.

PostHog is the sole recorder: 10% sampling and 30-day retention, all text and
inputs masked, images/video/canvas/iframes/private regions blocked, no console
capture or network headers/bodies. Link attributes and recording metadata URLs
have independent redaction hooks; JSON-LD is disabled. The `session-replay` flag
is an operational switch and does not override provider sampling.

The `website-cta-copy` experiment has a local control fallback and a benefit
variant pointing to the same install section. It and the optional first-value
rating survey remain provider drafts until production acceptance. Feedback is
gated by `product-feedback` and the successful installation-copy milestone.

Sentry captures redacted errors with stack coordinates/debug IDs, 10% sampled
traces, and bounded clipboard outcome logs/count/duration metrics. The v11 SDK's
`dataCollection` options disable user, cookie, header, body, query, local-variable,
AI, database and queue content. Final event/span/log/metric projections apply
independently. No tunnel or arbitrary external origin receives tracing headers.
The website server collects sanitized operational diagnostics independently of
browser preferences; browser telemetry never initializes on a privacy opt-out.

### Choices and failure isolation

The compact cookie notice and footer **Cookie settings** offer separate Analytics
and Advertising choices. Browser analytics, diagnostics and Vercel measurements
require Analytics consent. Global Privacy Control, Do Not Track and unavailable
local storage also prevent initialization. Opt-out stops
recording, clears the stored PostHog identity and suppresses all Sentry envelope
types at the transport boundary, including streamed spans. Another tab observes
the stored choice. Re-enabling analytics generates a new identity. Provider
failures do not change navigation or clipboard results. See
[website consent delivery](website-consent.md) for the released behavior,
tag mapping and production acceptance.

### Deployment and source maps

Production-only Vercel variables were configured and read back on October 20, 2026. The original checkout's ignored `secrets.md` records credentials and the
shared existing upload-token location; operator credentials never ship in clients.

```dotenv
NEXT_PUBLIC_POSTHOG_ENABLED=true
NEXT_PUBLIC_POSTHOG_KEY=<ingestion key>
NEXT_PUBLIC_SENTRY_ENABLED=true
NEXT_PUBLIC_SENTRY_DSN=<burrow-website DSN>
SENTRY_ENABLED=true
SENTRY_DSN=<burrow-website DSN>
SENTRY_AUTH_TOKEN=<server/build-only upload credential>
```

Both SDKs require a production Vercel build and their explicit enable switch.
Sentry's Next.js build plugin uploads source maps and deletes them after upload.
Enabled production builds require upload authorization and fail on upload errors.
Release names are `burrow-website@<exact Vercel Git SHA>` (GitHub SHA for CI).
The root product version is attached to analytics; no new version is set by this
feature. Use the existing Release Please/frozen promotion flow.

### Verification

Playwright serves the installed PostHog SDK extensions and intercepts both
providers. It recursively decodes rrweb compression, inspects actual Sentry v11
streamed envelopes, and verifies error/span/log/metric/event redaction, privacy
signals, identity reset, experiment exposure, successful copy ownership, failure
isolation, and masking. The test-only random source exercises sampled spans;
production retains normal 10% sampling. Test credentials are nonfunctional.

```sh
pnpm lint
pnpm typecheck:website
VERCEL_ENV=production NEXT_PUBLIC_POSTHOG_ENABLED=true NEXT_PUBLIC_POSTHOG_KEY=phc_playwright_only NEXT_PUBLIC_SENTRY_ENABLED=true NEXT_PUBLIC_SENTRY_DSN=https://00000000000000000000000000000000@o0.ingest.us.sentry.io/1 pnpm build
pnpm --filter website test
node --test scripts/release/*.test.mjs
bash scripts/ci/run-actionlint.sh
```

After protected deployment, verify exact release/source-map association, real
event/trace/log/metric ingestion, sampled masked recordings, flags and alert
routing before activating experiments or surveys.

The verified website commit `9e598d4dea3fdb722811a58d7da6fc7879d37370` uploaded
real Sentry source maps in a production-path build; release readback returned 200.
All nine browser tests pass, including real streamed Sentry envelopes.

## Go CLI, TUI and service

Browser opt-out also rotates a local identity generation. Every event and Sentry
envelope checks it, so re-enabling in one tab cannot revive another tab's cleared
identity. Preference-write failures stop collection in the affected document.
The real-SDK browser suite covers both cases.

The `protocol/telemetry` package exposes only closed event/operation/outcome enums
and monotonic timing. Bounded nonblocking queues isolate the SDKs from tunnel
work. Payload projection independently drops private/unknown fields at the SDK
boundary. Uploads have 500ms deadlines and shutdown waits at most 550ms. Expected
network/authorization failures are `unavailable`, while internal failures are
`error`. Sentry logs describe low-volume operations; forwarding uses metrics and
sampled traces without per-request text logs.

CLI owns `tunnel_opened`, `tunnel_reconnected`, and one
`tunnel_request_completed` after the first stream returns local response bytes.
The last event describes transport success, not an HTTP status or payload. Server
operations do not duplicate these activation milestones. The random CLI identity
is separate from owner credentials; server identity belongs to the service
installation, not to a connecting user. Client and server identities are not
correlated. No trace headers are propagated into management/tunnel traffic.

Only admitted production releases link public ingestion keys/DSNs and the exact
source revision. Local builds and CI are inert; server collection additionally
requires `BURROW_TELEMETRY_ENABLED=true`. The protected production GitHub
environment holds `BURROW_POSTHOG_KEY`, `BURROW_CLI_SENTRY_DSN`,
`BURROW_SERVER_SENTRY_DSN`, and server-only `SENTRY_AUTH_TOKEN`; all were configured
and read back October 20, 2026. The upload token is never a linker/build argument.
Source release preflight validates the checkout, ingestion credential types and
DSNs, creates exact-source Sentry releases, and verifies readback before publishing.
Go stacks retain file/function/line information and exact-commit GitHub links;
local filesystem prefixes, source excerpts, locals and error strings are removed.

`burrowctl telemetry disable`, `DO_NOT_TRACK=1`, and `BURROW_TELEMETRY=0` stop both
providers. Opt-out removes only the analytics UUID, preserving tunnel owner
credentials. Each upload rechecks the preference and identity generation, so
long-running commands cannot resume a cleared identity after re-enable. Private
storage failure fails closed. Preference/help/version commands emit no events.

Go tests exercise actual PostHog/Sentry wire payloads, independent SDK scopes,
concurrent first-launch identity/reset, opt-out of queued data, bounded provider
failure, and real local WebSocket/yamux forwarding with one first-value milestone.

## Released acceptance — 21 October 2026 UTC

Feature/fix PRs #34/#37, coordinated release #38 and frozen production #39 are
merged. Immutable **v0.3.7** identifies source
`cedd38fbfd038e38be149a4822dd9e5dcf21e705`; production is
`2e05cbc29b12e74410fad9ba0e0eaa4dd5adc0ad`. Protected deployment
[39517280388](https://github.com/miniaxolotl/burrow/actions/runs/39517280388)
passes on attempt 2 after npm propagation. All five published package hashes
verify. Historical v0.3.6 and its superseded frozen candidate remain retained.

The published Linux CLI opened a real HTTPS tunnel and forwarded two synthetic
requests. Durable opt-out preserves credentials, removes the analytics identity
and rotates it on re-enable. PostHog indexes 11 exact-release CLI events, exactly
one tunnel-open and first-forward milestone, and production server outcomes;
private fixture markers are absent. Sentry accepts CLI/server logs and metrics.
Its current **spans** dataset indexes two CLI spans and one server span for the
exact 0.3.7 source. Legacy `transaction_indexed` outcomes marked `discarded` reflect
the project's indexing feature, rather than failed span delivery.

The released website passes cross-tab opt-out/generation checks, live event
readback and decoded replay masking at the unchanged 10% sample. Recording
`01a15bf5-b96b-7ac7-a198-7f2b3795f01f` is available in the provider. Browser Sentry
event `1674e57c13404db4957b7d3b29c38d53` resolves through uploaded source maps to
`packages/website/src/lib/diagnostics.ts`.

All 12 dashboard queries execute successfully. Activation counts use anonymous
event identities by surface; the person-cohort is reference-only because PostHog
documents limitations for personless events. Provider readback finds zero person
profiles. Daily alerts and Monday reports remain enabled; CTA experiments and
milestone surveys remain drafts. Release gates passed Go/race/build checks,
11 browser cases, release contracts, workflow lint and infrastructure validation.
