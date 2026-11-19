import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";

const installer = readFileSync(
  new URL("../install.sh", import.meta.url),
  "utf8",
);
const reviewedInventory =
  "ea9c74f470ac11a70e74b8e9bd8c9ba4287a0cc7ed553c371d0dca47865cfe76";
const sha = (bytes) => createHash("sha256").update(bytes).digest("hex");

test("install only the reviewed platform archive, without replacing a working binary on failure", (t) => {
  const dir = mkdtempSync(join(tmpdir(), "burrow-install-"));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const bin = join(dir, "tools");
  mkdirSync(bin);
  for (const tool of [
    "awk",
    "chmod",
    "cp",
    "gzip",
    "mkdir",
    "mktemp",
    "mv",
    "rm",
    "tar",
  ]) {
    symlinkSync(
      execFileSync("which", [tool], { encoding: "utf8" }).trim(),
      join(bin, tool),
    );
  }
  const shim = (name, body) =>
    writeFileSync(join(bin, name), `#!/bin/sh\n${body}\n`, { mode: 0o755 });
  shim("uname", 'case "$1" in -s) echo "$OS";; -m) echo "$ARCH";; esac');
  // Source release v0.3.24 is assetless; GitHub latest points to v0.3.12.
  shim(
    "curl",
    `for arg do
    case "$arg" in
      https://*/releases/latest) echo 'Location: /releases/tag/v0.3.12'; exit 0;;
      https://github.com/miniaxolotl/burrow/releases/download/v0.3.23/"$ASSET") src=archive;;
      https://github.com/miniaxolotl/burrow/releases/download/v0.3.23/checksums.txt) src=inventory;;
      https://*) exit 22;;
      -o) output=next; continue;;
    esac
    if [ "$output" = next ]; then dest=$arg; output=; fi
  done
  cp "$HOME/$src" "$dest"`,
  );
  const installed = join(dir, ".local/bin/burrowd");
  mkdirSync(join(dir, ".local/bin"), { recursive: true });
  writeFileSync(join(dir, "burrowd"), "verified fixture");
  execFileSync("tar", ["-czf", "archive", "burrowd"], { cwd: dir });
  const digest = sha(readFileSync(join(dir, "archive")));
  const run = (os, arch, asset, manifest, failure, delivered = manifest) => {
    writeFileSync(installed, "previous");
    writeFileSync(join(dir, "inventory"), delivered);
    // Trust this disposable inventory in the copied script only.
    const { status, stderr } = spawnSync("/bin/sh", ["-s", "--", "--local"], {
      input: installer.replace(reviewedInventory, sha(manifest)),
      encoding: "utf8",
      env: {
        PATH: bin,
        HOME: dir,
        TMPDIR: dir,
        ASSET: asset,
        OS: os,
        ARCH: arch,
      },
    });
    if (failure) {
      assert.notEqual(status, 0);
      assert.match(stderr, failure);
    } else assert.equal(status, 0, stderr);
    assert.equal(
      readFileSync(installed, "utf8"),
      failure ? "previous" : "verified fixture",
    );
  };
  const asset = "burrowd_0.3.23_linux_amd64.tar.gz";
  const manifest = `${digest}  ${asset}\n`;
  run("Linux", "x86_64", asset, manifest, /sha256sum|shasum/);
  const hasher = process.platform === "darwin" ? "shasum" : "sha256sum";
  symlinkSync(
    execFileSync("which", [hasher], { encoding: "utf8" }).trim(),
    join(bin, hasher),
  );
  for (const [os, arch, suffix] of [
    ["Linux", "x86_64", "linux_amd64"],
    ["Linux", "aarch64", "linux_arm64"],
    ["Darwin", "arm64", "darwin_arm64"],
    ["Darwin", "x86_64", "darwin_amd64"],
  ]) {
    const name = `burrowd_0.3.23_${suffix}.tar.gz`;
    run(os, arch, name, `${digest}  ${name}\n`);
  }
  run("Linux", "amd64", asset, manifest, /checksum/i, "different inventory");
  run(
    "Linux",
    "amd64",
    asset,
    manifest.replace(asset, `other-${asset}`),
    /entry/i,
  );
  run("Linux", "amd64", asset, manifest.repeat(2), /entry/i);
  run(
    "Linux",
    "amd64",
    asset,
    manifest.replace(digest, "0".repeat(64)),
    /checksum/i,
  );
});
