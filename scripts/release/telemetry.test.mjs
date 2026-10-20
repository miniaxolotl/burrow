import assert from "node:assert/strict";
import test from "node:test";
import { validateTelemetryRelease } from "./telemetry.mjs";

test("release telemetry rejects mismatched source, operator keys and unsafe DSNs before publication", () => {
  const sha = "a".repeat(40);
  const env = {
    RELEASE_SOURCE_SHA: sha,
    BURROW_TELEMETRY_BUILD: "production",
    BURROW_POSTHOG_KEY: "phc_test",
    BURROW_CLI_SENTRY_DSN: `https://${"a".repeat(32)}@o1.ingest.us.sentry.io/1`,
    BURROW_SERVER_SENTRY_DSN: `https://${"b".repeat(32)}@o1.ingest.us.sentry.io/2`,
    SENTRY_AUTH_TOKEN: "server-only",
  };
  assert.deepEqual(
    validateTelemetryRelease(env, sha).map((release) => release.version),
    [`burrow-cli@${sha}`, `burrow-server@${sha}`],
  );
  for (const change of [
    { RELEASE_SOURCE_SHA: "b".repeat(40) },
    { BURROW_TELEMETRY_BUILD: "preview" },
    { BURROW_POSTHOG_KEY: "phx_operator" },
    { SENTRY_AUTH_TOKEN: "" },
    { BURROW_CLI_SENTRY_DSN: "https://credentials@private.example/1" },
    {
      BURROW_SERVER_SENTRY_DSN:
        env.BURROW_SERVER_SENTRY_DSN + "?secret=private",
    },
  ])
    assert.throws(() => validateTelemetryRelease({ ...env, ...change }, sha));
});
