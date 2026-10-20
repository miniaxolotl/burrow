/** Local image builds stay local; hosted publication is the protected Deploy workflow. */
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

if (process.env.GHCR_REGISTRY || process.env.DOCKERHUB_REGISTRY) {
  throw new Error(
    "Publish through Deploy on production; local deploy only builds burrowd:local",
  );
}
execFileSync(
  "docker",
  ["build", "--build-arg", "VERSION=local", "-t", "burrowd:local", "."],
  {
    cwd: fileURLToPath(new URL("../../../", import.meta.url)),
    stdio: "inherit",
  },
);
