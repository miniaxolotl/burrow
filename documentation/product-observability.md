# Product observability

## Website implementation

The website integrates PostHog US project **677269** and Sentry project
**burrow-website** in the existing `bramble-labs-inc` team. The CLI/server source
slice and production runtime acceptance are tracked separately; provider setup
alone does not mean those runtimes are instrumented.

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

The footer control disables both browser providers. Global Privacy Control, Do
Not Track and unavailable local storage also prevent initialization. Opt-out stops
recording, clears the stored PostHog identity and suppresses all Sentry envelope
types at the transport boundary, including streamed spans. Another tab observes
the stored choice. Re-enabling analytics generates a new identity. Provider
failures do not change navigation or clipboard results. Vercel's existing
cookie-free measurements remain separately documented in the public policy.

### Deployment and source maps

Production-only Vercel variables were configured and read back on October 20,
2026. The original checkout's ignored `secrets.md` records credentials and the
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
