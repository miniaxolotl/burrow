/**
 * Release script: publishes burrowctl npm packages and creates GitHub release.
 * Run with: pnpm --filter @script/release run release
 */

const { execSync } = await import("node:child_process");

const REPO = "miniaxolotl/burrow";

const GORELEASER_DIRS: Record<string, string> = {
  "linux-x64": "dist/burrowctl_linux_amd64_v1",
  "linux-arm64": "dist/burrowctl_linux_arm64",
  "darwin-x64": "dist/burrowctl_darwin_amd64_v1",
  "darwin-arm64": "dist/burrowctl_darwin_arm64",
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

async function getGithubRelease(tag: string): Promise<boolean> {
  try {
    execSync(`gh release view ${tag} --repo ${REPO}`, { stdio: "pipe" });
    return true;
  } catch {
    return false;
  }
}

async function createGithubRelease(tag: string, version: string) {
  const exists = await getGithubRelease(tag);
  if (exists) {
    console.log(`✓ Release ${tag} already exists on GitHub`);
    return;
  }

  const tagExists = execSync("git tag -l", { encoding: "utf8" })
    .split("\n")
    .map((t) => t.trim())
    .includes(tag);

  if (!tagExists) {
    run(`git tag -a ${tag} -m "Release ${tag}"`);
    run(`git push origin ${tag}`);
  }

  run(`gh release create ${tag} --generate-notes --repo ${REPO}`);
  console.log(`✓ GitHub release ${tag} created`);
}

async function publishPlatformPackages(version: string, dryRun: boolean) {
  for (const [platform, dir] of Object.entries(GORELEASER_DIRS)) {
    const pkgDir = `../../packages/burrowctl-${platform}`;
    const binSrc = `${dir}/burrowctl`;

    console.log(`--- ${platform} ---`);

    if (!dryRun) {
      run(`cp ${binSrc} ${pkgDir}/burrowctl`);
      run(`chmod +x ${pkgDir}/burrowctl`);

      const pkgJsonPath = `${pkgDir}/package.json`;
      const pkgJson = JSON.parse(execSync(`cat ${pkgJsonPath}`, { encoding: "utf8" }));
      pkgJson.version = version;
      execSync(`node -e "const fs=require('fs'); fs.writeFileSync('${pkgJsonPath}', JSON.stringify(${JSON.stringify(pkgJson)}, null, 2) + '\\n')"`);

      run(`cd ${pkgDir} && npm publish --access public`);
    }

    console.log(`✓ Published @miniaxolotl/burrowctl-${platform}@${version}`);
  }
}

async function publishMainPackage(version: string, dryRun: boolean) {
  const mainDir = "../../packages/burrowctl";
  const pkgJsonPath = `${mainDir}/package.json`;

  if (!dryRun) {
    const pkgJson = JSON.parse(execSync(`cat ${pkgJsonPath}`, { encoding: "utf8" }));
    pkgJson.version = version;
    for (const dep of Object.keys(pkgJson.optionalDependencies || {})) {
      pkgJson.optionalDependencies[dep] = version;
    }
    execSync(`node -e "const fs=require('fs'); fs.writeFileSync('${pkgJsonPath}', JSON.stringify(${JSON.stringify(pkgJson)}, null, 2) + '\\n')"`);

    run(`cd ${mainDir} && npm publish --access public`);
  }

  console.log(`✓ Published @miniaxolotl/burrowctl@${version}`);
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
      run("pnpm --filter @miniaxolotl/burrowctl build");

      await publishPlatformPackages(version, false);
      await publishMainPackage(version, false);

      console.log(`✓ Published @miniaxolotl/burrowctl@${version} to npm`);
    }
  }

  if (dryRun) {
    console.log("\n✓ Dry run complete — no changes published");
  } else {
    await createGithubRelease(`v${version}`, version);
    console.log("\n✓ Release complete");
  }
}

release().catch((err) => {
  console.error("Release failed:", err);
  process.exit(1);
});