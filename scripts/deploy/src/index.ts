/**
 * Local deploy script: builds and pushes Docker images to GHCR and/or Docker Hub.
 *
 * Registries are enabled by setting the corresponding env var:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl   → push to GitHub Container Registry
 *   DOCKERHUB_REGISTRY=miniaxolotl      → push to Docker Hub
 *
 * Run with: pnpm --filter @script/deploy run deploy
 *
 * Examples:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl DOCKERHUB_REGISTRY=miniaxolotl pnpm --filter @script/deploy run deploy
 */

const { execSync } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const IMAGE = "burrowd";
const ROOT = path.resolve(new URL("../../..", import.meta.url).pathname);

function run(cmd: string, cwd?: string) {
  console.log(`> ${cmd}`);
  execSync(cmd, { stdio: "inherit", cwd: cwd || ROOT });
}

async function deploy() {
  const packageJson = JSON.parse(
    fs.readFileSync(
      path.join(ROOT, "packages", "burrowctl", "package.json"),
      "utf8",
    ),
  );
  const version = packageJson.version;

  const registries: { name: string; url: string }[] = [];

  if (process.env.GHCR_REGISTRY) {
    registries.push({ name: "GHCR", url: process.env.GHCR_REGISTRY });
  }
  if (process.env.DOCKERHUB_REGISTRY) {
    registries.push({ name: "Docker Hub", url: process.env.DOCKERHUB_REGISTRY });
  }

  const tagSuffixes = ["latest", version];

  const allTags =
    registries.length === 0
      ? tagSuffixes.map((t) => `${IMAGE}:${t}`)
      : registries.flatMap(({ url }) =>
          tagSuffixes.map((t) => `${url}/${IMAGE}:${t}`),
        );

  console.log(`\n=== Deploy ${IMAGE}:${tagSuffixes.join(", ")} ===\n`);

  const tagFlags = allTags.map((t) => `-t ${t}`).join(" ");
  run(`docker build --build-arg VERSION=${version} ${tagFlags} .`);

  if (registries.length === 0) {
    console.log(
      `\n✓ Built ${allTags.join(", ")} (no registry set, skipping push)`,
    );
    return;
  }

  for (const tag of allTags) {
    run(`docker push ${tag}`);
  }

  console.log(`\n✓ Deployed to ${registries.map((r) => r.name).join(" + ")}`);
}

deploy().catch((err) => {
  console.error("Deploy failed:", err);
  process.exit(1);
});
