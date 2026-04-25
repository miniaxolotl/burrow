#!/usr/bin/env node
"use strict";

const { spawnSync } = require("child_process");
const { join, dirname } = require("path");

const PLATFORMS = {
  "linux-x64":    "@miniaxolotl/burrowctl-linux-x64",
  "linux-arm64":  "@miniaxolotl/burrowctl-linux-arm64",
  "darwin-x64":   "@miniaxolotl/burrowctl-darwin-x64",
  "darwin-arm64": "@miniaxolotl/burrowctl-darwin-arm64",
};

const key = `${process.platform}-${process.arch}`;
const pkgName = PLATFORMS[key];

if (!pkgName) {
  process.stderr.write(`burrowctl: unsupported platform ${key}\n`);
  process.exit(1);
}

let binDir;
try {
  binDir = dirname(require.resolve(`${pkgName}/package.json`));
} catch {
  process.stderr.write(`burrowctl: platform package ${pkgName} is not installed\n`);
  process.exit(1);
}

const result = spawnSync(
  join(binDir, "burrowctl"),
  process.argv.slice(2),
  { stdio: "inherit" },
);

process.exit(result.status ?? 1);
