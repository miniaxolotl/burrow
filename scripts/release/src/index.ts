/**
 * Release script: builds burrowctl with goreleaser and publishes to npm.
 * Run with: pnpm --filter @script/release run release
 */

const { execSync } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const BIN_PATHS = {
  "linux-x64": "burrowctl_linux_amd64_v1/burrowctl",
  "linux-arm64": "burrowctl_linux_arm64_v8.0/burrowctl",
  "darwin-x64": "burrowctl_darwin_amd64_v1/burrowctl",
  "darwin-arm64": "burrowctl_darwin_arm64_v8.0/burrowctl",
};

const ROOT = path.resolve(new URL("../../..", import.meta.url).pathname);
const DIST = path.join(ROOT, "dist");
const PKGS = path.join(ROOT, "packages");

function run(cmd, cwd, ignoreErrors = false) {
  console.log(`> ${cmd}`);
  try {
    execSync(cmd, { stdio: "inherit", cwd: cwd || ROOT });
  } catch (err) {
    if (!ignoreErrors) {
      throw err;
    }
  }
}

async function getNpmVersion(pkg) {
  try {
    const version = execSync(`npm view ${pkg} version --json`, {
      encoding: "utf8",
    }).trim();
    return version.replace(/"/g, "");
  } catch {
    return null;
  }
}

async function release() {
  const dryRun = process.argv.includes("--dry-run");

  console.log("\n=== Release ===\n");

  const pkgJsonPath = path.join(PKGS, "burrowctl", "package.json");
  const packageJson = JSON.parse(fs.readFileSync(pkgJsonPath, "utf8"));
  const version = packageJson.version;

  const npmVersion = await getNpmVersion("@miniaxolotl/burrowctl");

  if (npmVersion === version) {
    console.log(`@miniaxolotl/burrowctl@${version} already on npm`);
    return;
  }

  console.log(`\nLocal ${version} -> npm: ${npmVersion || "none"}\n`);

  if (dryRun) {
    console.log("\nDry run complete - no changes published");
    return;
  }

  // Build binaries with goreleaser
  run('export PATH="$PATH:$(go env GOPATH)/bin" && go install github.com/goreleaser/goreleaser/v2@v2', undefined, true);
  run('export PATH="$PATH:$(go env GOPATH)/bin" && goreleaser build --clean --snapshot --id burrowctl');

  // Verify binaries were built
  for (const [, binPath] of Object.entries(BIN_PATHS)) {
    const fullPath = path.join(DIST, binPath);
    if (!fs.existsSync(fullPath)) {
      throw new Error(`Binary not found: ${fullPath}`);
    }
  }

  // Publish platform-specific packages
  for (const [platform, binPath] of Object.entries(BIN_PATHS)) {
    const pkgDir = path.join(PKGS, `burrowctl-${platform}`);
    const pkgJsonPath2 = path.join(pkgDir, "package.json");

    console.log(`\n--- ${platform} ---`);

    fs.copyFileSync(path.join(DIST, binPath), path.join(pkgDir, "burrowctl"));
    fs.chmodSync(path.join(pkgDir, "burrowctl"), 0o755);

    const pkgJson = JSON.parse(fs.readFileSync(pkgJsonPath2, "utf8"));
    pkgJson.version = version;
    fs.writeFileSync(pkgJsonPath2, JSON.stringify(pkgJson, null, 2) + "\n");

    run(`npm publish --access public`, pkgDir);
    console.log(`Published @miniaxolotl/burrowctl-${platform}@${version}`);
  }

  // Publish main package
  const mainDir = path.join(PKGS, "burrowctl");
  const mainPkgJsonPath = path.join(mainDir, "package.json");
  const mainPkgJson = JSON.parse(fs.readFileSync(mainPkgJsonPath, "utf8"));
  mainPkgJson.version = version;
  fs.writeFileSync(mainPkgJsonPath, JSON.stringify(mainPkgJson, null, 2) + "\n");

  run(`npm publish --access public`, mainDir);
  console.log(`Published @miniaxolotl/burrowctl@${version}`);

  console.log("\nRelease complete");
}

release().catch((err) => {
  console.error("Release failed:", err);
  process.exit(1);
});
