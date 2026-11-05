import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { deploymentReady, waitForDeployment } from "../infra/wait-app.mjs";
const digest = `sha256:${"a".repeat(64)}`;
const spec = (value = digest) => ({
  services: [{ name: "burrowd", instance_count: 1, image: { digest: value } }],
});
const active = () => ({
  spec: spec(),
  active_deployment: { id: "deployment", phase: "ACTIVE", spec: spec() },
});

test("infrastructure readiness waits for the provider generation before public health", () => {
  const source = readFileSync("scripts/infra/apply.sh", "utf8");
  assert.ok(source.indexOf("node scripts/infra/wait-app.mjs") >= 0);
  assert.ok(
    source.indexOf("node scripts/infra/wait-app.mjs") <
      source.indexOf('"$url/health"'),
  );
});

test("provider readiness accepts only the active exact-image single-instance generation", () => {
  assert.equal(deploymentReady(active(), digest), true);
});

test("a desired image on the spec does not make the previous generation ready", () => {
  const app = active();
  app.active_deployment.spec = spec(`sha256:${"b".repeat(64)}`);
  assert.equal(deploymentReady(app, digest), false);
});

test("pending or non-active generations cannot pass readiness", () => {
  const app = active();
  app.in_progress_deployment = { id: "next", phase: "DEPLOYING" };
  assert.equal(deploymentReady(app, digest), false);
  delete app.in_progress_deployment;
  app.active_deployment.phase = "DEPLOYING";
  assert.equal(deploymentReady(app, digest), false);
});

test("provider readiness rejects changed desired images and unsafe replicas", () => {
  const app = active();
  app.spec = spec(`sha256:${"b".repeat(64)}`);
  assert.throws(() => deploymentReady(app, digest), /superseded/);
  app.spec = spec();
  app.active_deployment.spec.services[0].instance_count = 2;
  assert.throws(() => deploymentReady(app, digest), /single instance/);
});

test("readiness follows old, pending and finally active generations", async () => {
  const old = active();
  old.active_deployment.spec = spec(`sha256:${"b".repeat(64)}`);
  const pending = active();
  pending.in_progress_deployment = { id: "next", phase: "DEPLOYING" };
  const snapshots = [old, pending, active()];
  let time = 0;
  let reads = 0;
  assert.equal(
    await waitForDeployment("app", digest, {
      now: () => time,
      sleep: async (ms) => {
        time += ms;
      },
      getApp: async () => snapshots[reads++],
    }),
    "deployment",
  );
  assert.equal(reads, 3);
  assert.equal(time, 10_000);
});

test("the absolute deadline includes provider read time", async () => {
  let time = 0;
  let reads = 0;
  await assert.rejects(
    waitForDeployment("app", digest, {
      timeoutMs: 100,
      now: () => time,
      sleep: async (ms) => {
        time += ms;
      },
      getApp: async (remaining) => {
        assert.equal(remaining, 100);
        time += 101;
        reads++;
        return active();
      },
    }),
    /did not become active in time/,
  );
  assert.equal(reads, 1);
});

test("pending generations stop at the absolute deadline without another read", async () => {
  let time = 0;
  let reads = 0;
  await assert.rejects(
    waitForDeployment("app", digest, {
      timeoutMs: 25,
      now: () => time,
      sleep: async (ms) => {
        assert.equal(ms, 25);
        time += ms;
      },
      getApp: async () => {
        reads++;
        return {
          ...active(),
          in_progress_deployment: { id: "next", phase: "DEPLOYING" },
        };
      },
    }),
    /did not become active in time/,
  );
  assert.equal(reads, 1);
});

test("provider failures and superseded requests fail closed", async () => {
  const failed = active();
  failed.in_progress_deployment = { id: "next", phase: "ERROR" };
  await assert.rejects(
    waitForDeployment("app", digest, { getApp: async () => failed }),
    /Provider deployment failed/,
  );
  const superseded = active();
  superseded.spec = spec(`sha256:${"b".repeat(64)}`);
  await assert.rejects(
    waitForDeployment("app", digest, { getApp: async () => superseded }),
    /superseded/,
  );
});
