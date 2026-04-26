/**
 * Release script: builds burrowctl with goreleaser and publishes to npm.
 * Run with: pnpm --filter @script/release run release
 */

const { execSync } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const ROOT = path.resolve(new URL("../../..", import.meta.url).pathname);

const BIN_PATHS: Record<string, string> = {
  "linux-x64": "burrowctl_linux_amd64_v1/burrowctl",
  "linux-arm64": "burrowctl_linux_arm64_v8.0/burrowctl",
  "darwin-x64": "burrowctl_darwin_amd64_v1/burrowctl",
  "darwin-arm64": "burrowctl_darwin_arm64_v8.0/burrowctl",
};

function run(cmd: string, cwd?: string, ignoreErrors = false) {
  console.log(`> ${cmd}`);
  try {
    execSync(cmd, { stdio: "inherit", cwd: cwd || ROOT });
  } catch (err) {
    if (!ignoreErrors) {
      throw err;
    }
  }
}

async function getNpmVersion(pkg: string): Promise<string | null> {
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

  const packageJson = JSON.parse(
    fs.readFileSync(
      path.join(ROOT, "packages", "burrowctl", "package.json"),
      "utf8",
    ),
  );
  const version = packageJson.version;

  const npmVersion = await getNpmVersion("@miniaxolotl/burrowctl");

  if (npmVersion === version) {
    console.log(`✓ @miniaxolotl/burrowctl@${version} already on npm`);
    return;
  }

  console.log(`\nLocal ${version} → npm: ${npmVersion || "none"}\n`);

  if (dryRun) {
    console.log("\n✓ Dry run complete — no changes published");
    return;
  }

  run(
    'export PATH="$PATH:$(go env GOPATH)/bin" && go install github.com/goreleaser/goreleaser/v2@latest',
    undefined,
    true,
  );
  run(
    `export PATH="$PATH:$(go env GOPATH)/bin" && GORELEASER_CURRENT_TAG=v${version} goreleaser build --clean --id burrowctl --skip=validate`,
  );

  for (const [platform, binPath] of Object.entries(BIN_PATHS)) {
    const pkgDir = path.join(ROOT, "packages", `burrowctl-${platform}`);
    const pkgJsonPath = path.join(pkgDir, "package.json");

    console.log(`--- ${platform} ---`);

    run(`cp dist/${binPath} ${pkgDir}/burrowctl`);
    run(`chmod +x ${pkgDir}/burrowctl`);

    const pkgJson = JSON.parse(fs.readFileSync(pkgJsonPath, "utf8"));
    pkgJson.version = version;
    fs.writeFileSync(pkgJsonPath, JSON.stringify(pkgJson, null, 2) + "\n");

    run("npm publish --access public", pkgDir);
    console.log(`✓ Published @miniaxolotl/burrowctl-${platform}@${version}`);
  }

  const mainDir = path.join(ROOT, "packages", "burrowctl");
  const mainPkgJsonPath = path.join(mainDir, "package.json");
  const mainPkgJson = JSON.parse(fs.readFileSync(mainPkgJsonPath, "utf8"));
  mainPkgJson.version = version;
  for (const dep of Object.keys(mainPkgJson.optionalDependencies || {})) {
    mainPkgJson.optionalDependencies[dep] = version;
  }
  fs.writeFileSync(
    mainPkgJsonPath,
    JSON.stringify(mainPkgJson, null, 2) + "\n",
  );

  run("npm publish --access public", mainDir);
  console.log(`✓ Published @miniaxolotl/burrowctl@${version} to npm`);

  console.log("\n✓ Release complete");
}

release().catch((err) => {
  console.error("Release failed:", err);
  process.exit(1);
});
