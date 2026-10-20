/** Publish the four native packages, then the npm wrapper, from GoReleaser output. */
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { publishPackages } from "../publication.mjs";

const ROOT = fileURLToPath(new URL("../../../", import.meta.url));
const platforms = {
  "linux-x64": ["linux", "amd64"],
  "linux-arm64": ["linux", "arm64"],
  "darwin-x64": ["darwin", "amd64"],
  "darwin-arm64": ["darwin", "arm64"],
} as const;

type Artifact = {
  name: string;
  path: string;
  type: string;
  goos: string;
  goarch: string;
};

async function release() {
  const dryRun = process.argv.includes("--dry-run");
  if (
    !dryRun &&
    (process.env.GITHUB_ACTIONS !== "true" ||
      process.env.GITHUB_REF !== "refs/heads/production")
  ) {
    throw new Error(
      "Publish through the protected production Deploy workflow; local checks use --dry-run",
    );
  }
  const mainDirectory = path.join(ROOT, "packages/burrowctl");
  const mainPath = path.join(mainDirectory, "package.json");
  const main = JSON.parse(fs.readFileSync(mainPath, "utf8"));
  const version: string = main.version;

  const artifactsPath = path.join(ROOT, "dist/artifacts.json");
  if (!fs.existsSync(artifactsPath)) {
    execFileSync("goreleaser", ["build", "--clean", "--id", "burrowctl"], {
      cwd: ROOT,
      stdio: "inherit",
      env: { ...process.env, GORELEASER_CURRENT_TAG: `v${version}` },
    });
  }
  const artifacts: Artifact[] = JSON.parse(
    fs.readFileSync(artifactsPath, "utf8"),
  );

  const metadata = JSON.parse(
    fs.readFileSync(path.join(ROOT, "dist/metadata.json"), "utf8"),
  );
  if (metadata.version !== version) {
    throw new Error(
      `Built version ${metadata.version} does not match package version ${version}`,
    );
  }
  const source = execFileSync("git", ["rev-parse", "HEAD"], {
    cwd: ROOT,
    encoding: "utf8",
  }).trim();
  if (
    metadata.commit !== source ||
    (process.env.RELEASE_SOURCE_SHA &&
      source !== process.env.RELEASE_SOURCE_SHA)
  ) {
    throw new Error("Built source does not match the admitted release source");
  }

  const directories: string[] = [];
  for (const [platform, [goos, goarch]] of Object.entries(platforms)) {
    const artifact = artifacts.find(
      (item) =>
        item.type === "Binary" &&
        item.name === "burrowctl" &&
        item.goos === goos &&
        item.goarch === goarch,
    );
    if (!artifact) throw new Error(`Missing burrowctl binary for ${platform}`);
    const directory = path.join(ROOT, "packages", `burrowctl-${platform}`);
    const packagePath = path.join(directory, "package.json");
    const pkg = JSON.parse(fs.readFileSync(packagePath, "utf8"));
    if (pkg.version !== version)
      throw new Error(`${pkg.name} version does not match ${version}`);
    const binary = path.join(directory, "burrowctl");
    fs.copyFileSync(path.resolve(ROOT, artifact.path), binary);
    fs.chmodSync(binary, 0o755);
    directories.push(directory);
  }

  // pnpm pack resolves workspace ranges. Preflight every tarball before any write.
  await publishPackages([...directories, mainDirectory], dryRun);
}

try {
  await release();
} catch (error) {
  console.error("Release failed:", error);
  process.exitCode = 1;
}
