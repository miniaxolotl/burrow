import assert from "node:assert/strict";
import { test } from "node:test";
import { readFileSync } from "node:fs";

test("release proposals leave installed versions and Go stamps owned by their artifacts", () => {
  const extra =
    JSON.parse(readFileSync("release-please-config.json")).packages["."][
      "extra-files"
    ] ?? [];
  assert.deepEqual(extra, []);
});
