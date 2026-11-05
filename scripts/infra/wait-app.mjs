import assert from "node:assert/strict";
import { resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";
import { pathToFileURL } from "node:url";

export function deploymentReady(app, digest) {
  const desired = app.spec?.services;
  assert.equal(desired?.length, 1, "Expected the single Burrow service");
  assert.equal(desired[0].name, "burrowd");
  assert.equal(desired[0].image?.digest, digest, "Deployment was superseded");
  if (app.in_progress_deployment?.id) {
    assert.notEqual(
      app.in_progress_deployment.phase,
      "ERROR",
      "Provider deployment failed",
    );
    return false;
  }
  const active = app.active_deployment;
  if (!active?.id || active.phase !== "ACTIVE") return false;
  const services = active.spec?.services;
  if (services?.[0]?.image?.digest !== digest) return false;
  assert.equal(services.length, 1, "Expected a single instance service");
  assert.equal(services[0].name, "burrowd");
  assert.equal(services[0].instance_count, 1, "Expected a single instance");
  return true;
}

export async function waitForDeployment(appId, digest, options = {}) {
  const now = options.now ?? Date.now;
  const sleep = options.sleep ?? delay;
  const deadline = now() + (options.timeoutMs ?? 300_000);
  const getApp =
    options.getApp ??
    (async (remaining) => {
      assert(
        process.env.DIGITALOCEAN_ACCESS_TOKEN,
        "Missing provider credential",
      );
      const response = await fetch(
        `https://api.digitalocean.com/v2/apps/${appId}`,
        {
          headers: {
            authorization: `Bearer ${process.env.DIGITALOCEAN_ACCESS_TOKEN}`,
          },
          signal: AbortSignal.timeout(Math.min(10_000, remaining)),
        },
      );
      assert(response.ok, `Provider read failed (${response.status})`);
      return (await response.json()).app;
    });
  while (now() < deadline) {
    const app = await getApp(deadline - now());
    if (now() <= deadline && deploymentReady(app, digest))
      return app.active_deployment.id;
    const remaining = deadline - now();
    if (remaining > 0) await sleep(Math.min(5_000, remaining));
  }
  throw new Error(
    "Exact-image provider generation did not become active in time",
  );
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  const [appId, digest] = process.argv.slice(2);
  assert.match(appId ?? "", /^[a-f0-9-]{36}$/);
  assert.match(digest ?? "", /^sha256:[a-f0-9]{64}$/);
  waitForDeployment(appId, digest)
    .then((id) => console.log(`Exact-image provider generation active: ${id}`))
    .catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
}
