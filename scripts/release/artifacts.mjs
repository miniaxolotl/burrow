import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import {
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const names = [
  "burrowctl",
  "burrowctl-linux-x64",
  "burrowctl-linux-arm64",
  "burrowctl-darwin-x64",
  "burrowctl-darwin-arm64",
];

export function verifyPackedArtifacts(artifacts) {
  assert.equal(
    artifacts.length,
    names.length,
    "Expected all five packaged artifacts",
  );
  const directory = mkdtempSync(
    join(process.env.CI ? tmpdir() : "/tmp/opencode", "burrow-artifact-smoke-"),
  );
  try {
    const packages = new Map();
    for (const name of names) {
      const qualified = `@miniaxolotl/${name}`;
      const matches = artifacts.filter((artifact) =>
        artifact.name.startsWith(`${qualified}@`),
      );
      assert.equal(matches.length, 1, `Missing or duplicated ${qualified}`);
      const destination = join(directory, "node_modules", qualified);
      mkdirSync(destination, { recursive: true });
      execFileSync("tar", [
        "-xzf",
        matches[0].archive,
        "--strip-components=1",
        "-C",
        destination,
      ]);
      const pkg = JSON.parse(
        readFileSync(join(destination, "package.json"), "utf8"),
      );
      assert.equal(`${pkg.name}@${pkg.version}`, matches[0].name);
      packages.set(name, pkg);
      const executable = join(
        destination,
        name === "burrowctl" ? "bin/burrowctl.js" : "burrowctl",
      );
      const mode = lstatSync(executable);
      assert.ok(
        mode.isFile() && (mode.mode & 0o111) !== 0,
        `${qualified} archive must contain an executable`,
      );
    }
    const wrapper = packages.get("burrowctl");
    for (const name of names.slice(1)) {
      assert.equal(packages.get(name).version, wrapper.version);
      assert.equal(
        wrapper.optionalDependencies[`@miniaxolotl/${name}`],
        wrapper.version,
      );
    }
    const output = execFileSync(
      process.execPath,
      [
        join(directory, "node_modules/@miniaxolotl/burrowctl/bin/burrowctl.js"),
        "--version",
      ],
      { encoding: "utf8", timeout: 10000 },
    );
    assert.equal(
      output.trim(),
      `burrowctl version ${wrapper.version}`,
      "Packaged native build version mismatch",
    );
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
}
