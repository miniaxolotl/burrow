/** Publish the four native packages, then the npm wrapper, from GoReleaser output. */
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

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

function published(name: string, version: string): boolean {
  try {
    return (
      JSON.parse(
        execFileSync(
          "npm",
          ["view", `${name}@${version}`, "version", "--json"],
          {
            encoding: "utf8",
            stdio: ["ignore", "pipe", "pipe"],
          },
        ),
      ) === version
    );
  } catch (error) {
    const stderr = String((error as { stderr?: Buffer }).stderr ?? "");
    if (/E404|No match found for version/.test(stderr)) return false;
    throw error;
  }
}

function publish(directory: string, name: string, version: string) {
  if (published(name, version)) {
    console.log(`${name}@${version} already published`);
    return;
  }
  const args = ["publish", "--access", "public"];
  if (process.env.GITHUB_ACTIONS === "true") args.push("--provenance");
  execFileSync("npm", args, { cwd: directory, stdio: "inherit" });
}

function release() {
  const mainDirectory = path.join(ROOT, "packages/burrowctl");
  const mainPath = path.join(mainDirectory, "package.json");
  const main = JSON.parse(fs.readFileSync(mainPath, "utf8"));
  const version: string = main.version;
  if (process.argv.includes("--dry-run")) {
    for (const suffix of [...Object.keys(platforms), ""]) {
      const name = suffix ? `${main.name}-${suffix}` : main.name;
      console.log(
        `${name}@${version}: ${published(name, version) ? "published" : "pending"}`,
      );
    }
    return;
  }

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
    publish(directory, pkg.name, version);
  }

  // npm does not resolve pnpm workspace ranges when publishing directly.
  const original = fs.readFileSync(mainPath, "utf8");
  try {
    for (const dependency of Object.keys(main.optionalDependencies)) {
      main.optionalDependencies[dependency] = version;
    }
    fs.writeFileSync(mainPath, JSON.stringify(main, null, 2) + "\n");
    publish(mainDirectory, main.name, version);
  } finally {
    fs.writeFileSync(mainPath, original);
  }
}

try {
  release();
} catch (error) {
  console.error("Release failed:", error);
  process.exitCode = 1;
}
