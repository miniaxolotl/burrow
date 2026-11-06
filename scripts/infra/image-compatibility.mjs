import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

// Permanent minimum for this deployment path, not an operator-supplied flag.
// Once persistent owner records exist a pre-fencing binary is never safe.
export function attachmentCompatible(labels) {
  assert.equal(
    labels?.["io.bramble.burrow.attachment-state"],
    "1",
    "Image lacks the required persistent attachment fencing; downgrade refused",
  );
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  const digest = process.argv[2];
  assert.match(digest ?? "", /^sha256:[a-f0-9]{64}$/);
  const image = `ghcr.io/miniaxolotl/burrowd@${digest}`;
  const options = {
    encoding: "utf8",
    timeout: 120_000,
    maxBuffer: 1024 * 1024,
  };
  execFileSync("docker", ["pull", image], options);
  const labels = JSON.parse(
    execFileSync(
      "docker",
      ["image", "inspect", image, "--format", "{{json .Config.Labels}}"],
      options,
    ),
  );
  attachmentCompatible(labels);
}
