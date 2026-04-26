/**
 * Deploy script: builds and pushes multi-platform Docker image to GHCR and/or Docker Hub.
 *
 * Registries are enabled by setting the corresponding env var:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl   -> push to GitHub Container Registry
 *   DOCKERHUB_REGISTRY=miniaxolotl      -> push to Docker Hub
 *
 * Run with: pnpm --filter @script/deploy run deploy
 *
 * Examples:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 *   DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 */

const { execSync } = await import("node:child_process");

const IMAGE = "burrowd";
const PLATFORMS = process.env.PLATFORMS || "linux/amd64,linux/arm64";
const ROOT = new URL("../../..", import.meta.url).pathname;

function run(cmd, cwd) {
  console.log(`> ${cmd}`);
  execSync(cmd, { stdio: "inherit", cwd: cwd || ROOT });
}

async function gitRevision() {
  return execSync("git rev-parse HEAD", { cwd: ROOT }).toString().trim();
}

async function buildLabels(tag) {
  const version = tag.replace(/^v/, "");
  const revision = await gitRevision();
  const created = new Date().toISOString();

  return [
    "--label", `org.opencontainers.image.version=${version}`,
    "--label", `org.opencontainers.image.revision=${revision}`,
    "--label", `org.opencontainers.image.created=${created}`,
  ];
}

async function registryLogin(registry, token, user) {
  console.log(`Logging in to ${registry}...`);
  run(`echo "${token}" | docker login ${registry} -u ${user} --password-stdin`);
}

async function deploy() {
  const tag = process.env.TAG || "latest";
  const tags = tag === "latest" ? ["latest"] : [tag, tag.replace(/^v/, "")];

  const registries = [];

  if (process.env.GHCR_REGISTRY) {
    registries.push({ name: "GHCR", url: process.env.GHCR_REGISTRY });
  }
  if (process.env.DOCKERHUB_REGISTRY) {
    registries.push({ name: "Docker Hub", url: process.env.DOCKERHUB_REGISTRY });
  }

  const ghToken = process.env.GH_TOKEN || process.env.GITHUB_TOKEN;

  if (process.env.GHCR_REGISTRY && ghToken) {
    const ghUser = process.env.GHCR_USER || "miniaxolotl";
    await registryLogin(process.env.GHCR_REGISTRY, ghToken, ghUser);
  }

  if (process.env.DOCKERHUB_REGISTRY && process.env.DOCKERHUB_TOKEN) {
    const dockerUser = process.env.DOCKERHUB_USER || process.env.DOCKERHUB_REGISTRY;
    await registryLogin(process.env.DOCKERHUB_REGISTRY, process.env.DOCKERHUB_TOKEN, dockerUser);
  }

  console.log(`\n=== Deploy ${IMAGE}:${tags.join(", ")} ===\n`);

  for (const registry of registries) {
    console.log(`\n--- Pushing to ${registry.name} ---`);

    for (const t of tags) {
      const fullImage = `${registry.url}/${IMAGE}:${t}`;
      const labels = await buildLabels(t);
      run(`docker buildx build --platform ${PLATFORMS} ${labels.join(" ")} -t ${fullImage} --push .`);
      console.log(`Pushed ${fullImage}`);
    }
  }

  if (registries.length === 0) {
    const labels = await buildLabels(tags[0]);
    const tagArgs = tags.map((t) => `-t ${IMAGE}:${t}`).join(" ");
    const platform = PLATFORMS.includes(",") && process.env.CI !== "true"
      ? "linux/amd64"
      : PLATFORMS;
    run(`docker buildx build --platform ${platform} ${labels.join(" ")} ${tagArgs} --load .`);
    console.log(
      `\nBuilt ${tags.map((t) => `${IMAGE}:${t}`).join(", ")} (no registry set, skipping push)`,
    );
  } else {
    console.log(`\nDeployed to ${registries.map((r) => r.name).join(" + ")}`);
  }
}

deploy().catch((err) => {
  console.error("Deploy failed:", err);
  process.exit(1);
});
