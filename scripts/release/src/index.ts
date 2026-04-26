/**
 * Release script: builds burrowctl with goreleaser and publishes to npm.
 * Run with: pnpm --filter @script/release run release
 */

const { execSync } = await import("node:child_process");
const fs = await import("node:fs");

const BIN_PATHS: Record<string, string> = {
  "linux-x64": "burrowctl_linux_amd64_v1/burrowctl",
  "linux-arm64": "burrowctl_linux_arm64_v1/burrowctl",
  "darwin-x64": "burrowctl_darwin_amd64_v1/burrowctl",
  "darwin-arm64": "burrowctl_darwin_arm64_v1/burrowctl",
};

async function run(cmd: string, ignoreErrors = false) {
  console.log(`> ${cmd}`);
  try {
    execSync(cmd, { stdio: "inherit" });
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
    execSync("cat ../../packages/burrowctl/package.json", { encoding: "utf8" }),
  );
  const version = packageJson.version;

  const npmVersion = await getNpmVersion("@miniaxolotl/burrowctl");

  if (npmVersion === version) {
    console.log(`✓ @miniaxolotl/burrowctl@${version} already deployed to npm`);
  } else {
    console.log(`\nLocal ${version} → npm: ${npmVersion || "none"}\n`);

    if (!dryRun) {
      run('export PATH="$PATH:$(go env GOPATH)/bin" && go install github.com/goreleaser/goreleaser/v2@v2 && goreleaser build --clean --snapshot --id burrowctl');

      for (const [platform, binPath] of Object.entries(BIN_PATHS)) {
        const pkgDir = `../../packages/burrowctl-${platform}`;
        const pkgJsonPath = `${pkgDir}/package.json`;

        console.log(`--- ${platform} ---`);

        run(`cp dist/${binPath} ${pkgDir}/burrowctl`);
        run(`chmod +x ${pkgDir}/burrowctl`);

        const pkgJson = JSON.parse(execSync(`cat ${pkgJsonPath}`, { encoding: "utf8" }));
        pkgJson.version = version;
        const tmpPath = `/tmp/pkg-json-${Date.now()}.json`;
        fs.writeFileSync(tmpPath, JSON.stringify(pkgJson, null, 2) + "\n");
        run(`cp ${tmpPath} ${pkgJsonPath}`);

        run(`cd ${pkgDir} && npm publish --access public`);
        console.log(`✓ Published @miniaxolotl/burrowctl-${platform}@${version}`);
      }

      const mainDir = "../../packages/burrowctl";
      const mainPkgJsonPath = `${mainDir}/package.json`;
      const mainPkgJson = JSON.parse(execSync(`cat ${mainPkgJsonPath}`, { encoding: "utf8" }));
      mainPkgJson.version = version;
      for (const dep of Object.keys(mainPkgJson.optionalDependencies || {})) {
        mainPkgJson.optionalDependencies[dep] = version;
      }
      const mainTmpPath = `/tmp/pkg-json-main-${Date.now()}.json`;
      fs.writeFileSync(mainTmpPath, JSON.stringify(mainPkgJson, null, 2) + "\n");
      run(`cp ${mainTmpPath} ${mainPkgJsonPath}`);

      run(`cd ${mainDir} && npm publish --access public`);
      console.log(`✓ Published @miniaxolotl/burrowctl@${version} to npm`);
    }
  }

  if (dryRun) {
    console.log("\n✓ Dry run complete — no changes published");
  } else {
    console.log("\n✓ Release complete");
  }
}

release().catch((err) => {
  console.error("Release failed:", err);
  process.exit(1);
});
