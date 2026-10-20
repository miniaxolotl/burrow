import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import {
  copyFileSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { verifyPackedArtifacts } from "./artifacts.mjs";
import { packArtifact } from "./pack.mjs";

test("packed native packages retain executable bits and the shipped wrapper launches the matching version", () => {
  const originalUmask = process.umask(0o022);
  const base = process.env.CI ? tmpdir() : "/tmp/opencode";
  const temporary = mkdtempSync(join(base, "burrow-packed-test-"));
  const version = "9.8.7";
  try {
    const artifacts = [];
    for (const name of [
      "burrowctl-linux-x64",
      "burrowctl-linux-arm64",
      "burrowctl-darwin-x64",
      "burrowctl-darwin-arm64",
      "burrowctl",
    ]) {
      const source = new URL(`../../packages/${name}/`, import.meta.url);
      const pkg = JSON.parse(
        readFileSync(new URL("package.json", source), "utf8"),
      );
      pkg.version = version;
      if (pkg.optionalDependencies)
        for (const key of Object.keys(pkg.optionalDependencies))
          pkg.optionalDependencies[key] = version;
      const directory = join(temporary, name);
      mkdirSync(directory);
      writeFileSync(join(directory, "package.json"), JSON.stringify(pkg));
      if (name === "burrowctl") {
        mkdirSync(join(directory, "bin"));
        copyFileSync(
          new URL("bin/burrowctl.js", source),
          join(directory, "bin/burrowctl.js"),
        );
      } else {
        writeFileSync(
          join(directory, "burrowctl"),
          `#!/bin/sh\nprintf '%s\\n' 'burrowctl version ${version}'\n`,
          { mode: 0o755 },
        );
      }
      const archive = packArtifact(directory, join(temporary, "packed", name));
      if (name === "burrowctl") {
        process.umask(0o077);
        pkg.optionalDependencies = Object.fromEntries(
          Object.entries(pkg.optionalDependencies).reverse(),
        );
        writeFileSync(join(directory, "package.json"), JSON.stringify(pkg));
        const repeated = packArtifact(directory, join(temporary, "reordered"));
        assert.deepEqual(
          readFileSync(archive),
          readFileSync(repeated),
          "dependency ordering must not change published bytes",
        );
      }
      artifacts.push({
        name: `${pkg.name}@${version}`,
        archive,
      });
    }
    assert.doesNotThrow(() => verifyPackedArtifacts(artifacts));
    assert.throws(
      () => verifyPackedArtifacts(artifacts.slice(1)),
      /five|Missing/,
    );
    const native = join(temporary, "burrowctl-darwin-arm64");
    const unpacked = join(temporary, "corrupt");
    mkdirSync(unpacked);
    execFileSync("tar", ["-xzf", artifacts[3].archive, "-C", unpacked]);
    execFileSync("chmod", ["644", join(unpacked, "package/burrowctl")]);
    const corrupt = join(native, "corrupt.tgz");
    execFileSync("tar", ["-czf", corrupt, "-C", unpacked, "package"]);
    assert.throws(
      () =>
        verifyPackedArtifacts(
          artifacts.map((artifact, index) =>
            index === 3 ? { ...artifact, archive: corrupt } : artifact,
          ),
        ),
      /executable/,
    );
  } finally {
    process.umask(originalUmask);
    rmSync(temporary, { recursive: true, force: true });
  }
});
