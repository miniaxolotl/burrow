import assert from "node:assert/strict";
import { test } from "node:test";
import { readFileSync } from "node:fs";
import * as admission from "./promotion.mjs";

test("release proposals leave installed versions and Go stamps owned by their artifacts", () => {
  const extra =
    JSON.parse(readFileSync("release-please-config.json")).packages["."][
      "extra-files"
    ] ?? [];
  assert.deepEqual(extra, []);
});

const receipt = {
  tag: "v0.3.24",
  production: "a".repeat(40),
  reviewedSource: "b".repeat(40),
};
const inputs = {
  production: receipt.production,
  paths: ["package.json", "CHANGELOG.md"],
  previous: { private: true, version: "0.3.23" },
  current: { private: true, version: "0.3.24" },
};
test("site delivery admits only the reviewed tree plus release metadata and tooling", () => {
  assert.equal(typeof admission.assertSiteOnly, "function");
  assert.doesNotThrow(() => admission.assertSiteOnly(receipt, inputs));
  for (const drift of [
    { production: "c".repeat(40) },
    { current: { ...inputs.current, scripts: { install: "changed" } } },
    ...[
      "packages/burrowctl/package.json",
      "packages/burrowctl-linux-arm64/package.json",
      "burrowd/version.go",
      "protocol/go.mod",
      "scripts/install.sh",
      "packages/website/src/app/page.tsx",
      "pnpm-lock.yaml",
      ".github/workflows/deploy.yml",
      "scripts/release/policy.json",
    ].map((path) => ({ paths: [path] })),
  ]) {
    assert.throws(() =>
      admission.assertSiteOnly(receipt, { ...inputs, ...drift }),
    );
  }
});
