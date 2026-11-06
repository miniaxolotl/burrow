import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

// Workspace sums can hide an incomplete standalone module (notably Docker's
// server-only context). Every pinned external requirement needs its own module
// checksum in each module that declares it, not only in a sibling go.sum.
for (const module of ["burrowd", "burrowctl", "protocol"]) {
  test(`${module} retains standalone requirement module checksums`, () => {
    const manifest = readFileSync(`${module}/go.mod`, "utf8");
    const sums = new Set(
      readFileSync(`${module}/go.sum`, "utf8")
        .trim()
        .split("\n")
        .map((line) => line.split(" ").slice(0, 2).join(" ")),
    );
    for (const [, name, version] of manifest.matchAll(
      /^\s+([^\s]+) (v[^\s]+)(?:\s|$)/gm,
    )) {
      if (name === "burrow/protocol") continue; // local replace, no remote sum
      assert.ok(
        sums.has(`${name} ${version}/go.mod`),
        `${module}/go.sum lacks ${name} ${version}/go.mod`,
      );
    }
  });
}
