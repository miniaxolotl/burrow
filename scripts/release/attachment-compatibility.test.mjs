import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { attachmentCompatible } from "../infra/image-compatibility.mjs";

test("persistent attachment rollout rejects pre-fencing and unknown images", () => {
  for (const labels of [
    null,
    {},
    { "io.bramble.burrow.attachment-state": "0" },
    { "io.bramble.burrow.attachment-state": "2" },
  ]) {
    assert.throws(() => attachmentCompatible(labels), /attachment/);
  }
  attachmentCompatible({ "io.bramble.burrow.attachment-state": "1" });
});

test("every infrastructure apply verifies the immutable image before planning", () => {
  const source = readFileSync("scripts/infra/apply.sh", "utf8");
  assert.ok(
    source.indexOf(
      'node scripts/infra/image-compatibility.mjs "$TF_VAR_image_digest"',
    ) > 0,
  );
  assert.ok(
    source.indexOf("image-compatibility.mjs") <
      source.indexOf('tofu -chdir="$root" plan'),
  );
  assert.match(
    readFileSync("Dockerfile", "utf8"),
    /io\.bramble\.burrow\.attachment-state="1"/,
  );
});
