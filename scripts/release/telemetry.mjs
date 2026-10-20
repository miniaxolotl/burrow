import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

export function validateTelemetryRelease(env, sourceSha) {
  if (env.RELEASE_SOURCE_SHA !== sourceSha || !/^[a-f0-9]{40}$/.test(sourceSha))
    throw new Error("Telemetry release must match admitted checkout");
  if (
    env.BURROW_TELEMETRY_BUILD !== "production" ||
    !/^phc_[a-zA-Z0-9_]+$/.test(env.BURROW_POSTHOG_KEY ?? "")
  )
    throw new Error("Production ingestion configuration missing");
  for (const key of ["BURROW_CLI_SENTRY_DSN", "BURROW_SERVER_SENTRY_DSN"]) {
    const dsn = new URL(env[key] ?? "");
    if (
      dsn.protocol !== "https:" ||
      !dsn.hostname.endsWith(".ingest.us.sentry.io") ||
      !/^[a-f0-9]{32}$/.test(dsn.username) ||
      !/^\/\d+$/.test(dsn.pathname) ||
      dsn.password ||
      dsn.search ||
      dsn.hash
    )
      throw new Error(`Invalid public DSN: ${key}`);
  }
  if (!env.SENTRY_AUTH_TOKEN) throw new Error("Release authorization missing");
  return ["cli", "server"].map((surface) => ({
    project: `burrow-${surface}`,
    version: `burrow-${surface}@${sourceSha}`,
    url: `https://github.com/miniaxolotl/burrow/commit/${sourceSha}`,
  }));
}

async function main() {
  const sha = execFileSync("git", ["rev-parse", "HEAD"], {
    encoding: "utf8",
  }).trim();
  const releases = validateTelemetryRelease(process.env, sha);
  const api = async (path, init = {}) => {
    const response = await fetch(`https://us.sentry.io/api/0/${path}`, {
      ...init,
      headers: {
        Authorization: `Bearer ${process.env.SENTRY_AUTH_TOKEN}`,
        "content-type": "application/json",
      },
      signal: AbortSignal.timeout(20_000),
    });
    if (!response.ok)
      throw new Error(`Sentry release request failed: ${response.status}`);
    return response.json();
  };
  for (const release of releases) {
    await api("organizations/bramble-labs-inc/releases/", {
      method: "POST",
      body: JSON.stringify({
        version: release.version,
        projects: [release.project],
        url: release.url,
        dateReleased: new Date().toISOString(),
      }),
    });
    const readback = await api(
      `projects/bramble-labs-inc/${release.project}/releases/${release.version}/`,
    );
    if (readback.version !== release.version || readback.url !== release.url)
      throw new Error("Sentry release readback mismatch");
    console.log(`Verified ${release.version}`);
  }
}
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href)
  await main();
