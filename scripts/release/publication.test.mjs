import assert from "node:assert/strict";
import test from "node:test";
import { publicationPlan, publishPlan } from "./publication.mjs";

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
