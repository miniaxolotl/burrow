/**
 * Deploy script: builds and pushes multi-platform Docker image to GHCR and/or Docker Hub,
 * then creates a GitHub release.
 *
 * Registries are enabled by setting the corresponding env var:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl   -> push to GitHub Container Registry
 *   DOCKERHUB_REGISTRY=miniaxolotl      -> push to Docker Hub
 *
 * GitHub release is created when GH_TOKEN or GITHUB_TOKEN is set.
 *
 * Run with: pnpm --filter @script/deploy run deploy
 *
 * Examples:
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 *   DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 *   GHCR_REGISTRY=ghcr.io/miniaxolotl DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 pnpm --filter @script/deploy run deploy
 */

const { execSync } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const IMAGE = "burrowd";
const REPO = "miniaxolotl/burrow";
const PLATFORMS = process.env.PLATFORMS || "linux/amd64,linux/arm64";
const ROOT = path.resolve(new URL("../../..", import.meta.url).pathname);

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

async function createGithubRelease(tag) {
  const token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN;
  if (!token) {
    console.log("No GH_TOKEN or GITHUB_TOKEN - skipping GitHub release");
    return;
  }

  console.log(`\n--- Creating GitHub release ${tag} ---`);

  const exists = execSync(`git tag -l "${tag}"`, { cwd: ROOT }).toString().trim();
  if (!exists) {
    run(`git tag -a ${tag} -m "Release ${tag}"`);
    run(`git push origin ${tag}`);
  }

  run(`gh release create ${tag} --generate-notes --repo ${REPO}`);
  console.log(`GitHub release ${tag} created`);
}

async function deploy() {
  const packageJson = JSON.parse(
    fs.readFileSync(path.join(ROOT, "packages", "burrowctl", "package.json"), "utf8"),
  );
  const version = packageJson.version;

  const tag = process.env.TAG || "latest";
  const versionTag = tag.replace(/^v/, "");
  const tags =
    tag === "latest"
      ? ["latest", `v${version}`]
      : ([tag, versionTag, versionTag === version ? "latest" : null].filter(
          Boolean,
        ));

  const registries = [];

  if (process.env.GHCR_REGISTRY) {
    registries.push({ name: "GHCR", url: process.env.GHCR_REGISTRY });
  }
  if (process.env.DOCKERHUB_REGISTRY) {
    registries.push({
      name: "Docker Hub",
      url: process.env.DOCKERHUB_REGISTRY,
    });
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

  if (tag !== "latest") {
    await createGithubRelease(tag);
  }
}

deploy().catch((err) => {
  console.error("Deploy failed:", err);
  process.exit(1);
});
