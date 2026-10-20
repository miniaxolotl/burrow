import assert from "node:assert/strict";
import test from "node:test";
import {
  publicationPlan,
  publishPlan,
  waitForPublication,
} from "./publication.mjs";

test("publication waits for registry propagation without republishing", async () => {
  let reads = 0;
  let waits = 0;
  await waitForPublication(
    { name: "native@1.0.0", integrity: "one" },
    {
      lookup: () => (++reads === 3 ? "one" : null),
      wait: async () => {
        waits++;
      },
      attempts: 3,
    },
  );
  assert.equal(reads, 3);
  assert.equal(waits, 2);
});

test("publication readback bounds missing results and rejects conflicts and provider errors immediately", async () => {
  const artifact = { name: "native@1.0.0", integrity: "one" };
  let reads = 0;
  let waits = 0;
  const wait = async () => {
    waits++;
  };
  await assert.rejects(
    waitForPublication(artifact, {
      lookup: () => {
        reads++;
        return null;
      },
      wait,
      attempts: 3,
    }),
    /not visible/,
  );
  assert.equal(reads, 3);
  assert.equal(waits, 2);
  const failWait = async () => {
    assert.fail("must not retry conflicts or provider errors");
  };
  await assert.rejects(
    waitForPublication(artifact, {
      lookup: () => "different",
      wait: failWait,
    }),
    /differs/,
  );
  await assert.rejects(
    waitForPublication(artifact, {
      lookup: () => {
        throw new Error("registry unavailable");
      },
      wait: failWait,
    }),
    /registry unavailable/,
  );
});

test("all artifacts are checked before the first publish, including on retry", async () => {
  const artifacts = [
    { name: "native", integrity: "one" },
    { name: "wrapper", integrity: "two" },
  ];
  const writes = [];
  await assert.rejects(
    async () =>
      publishPlan(
        await publicationPlan(artifacts, async (name) =>
          name === "native" ? null : "wrong",
        ),
        (pkg) => writes.push(pkg),
      ),
    /different contents/,
  );
  assert.deepEqual(writes, []);
  const plan = await publicationPlan(artifacts, async (name) =>
    name === "native" ? "one" : null,
  );
  await publishPlan(plan, async (pkg) => writes.push(pkg.name));
  assert.deepEqual(writes, ["wrapper"]);
});

test("provider failure and partial publication stop and resume without overwrites", async () => {
  const artifacts = [
    { name: "native", integrity: "one" },
    { name: "wrapper", integrity: "two" },
  ];
  await assert.rejects(
    publicationPlan(artifacts, async () => {
      throw new Error("registry unavailable");
    }),
    /registry unavailable/,
  );
  const published = new Map();
  const publish = async (pkg) => {
    if (pkg.name === "wrapper") throw new Error("response lost");
    published.set(pkg.name, pkg.integrity);
  };
  await assert.rejects(
    publishPlan(await publicationPlan(artifacts, async () => null), publish),
    /response lost/,
  );
  const plan = await publicationPlan(artifacts, async (name) =>
    published.get(name),
  );
  assert.deepEqual(
    plan.map((item) => item.name),
    ["wrapper"],
  );
});
