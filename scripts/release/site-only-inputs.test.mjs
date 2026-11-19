import assert from "node:assert/strict";
import { test } from "node:test";
import { execFileSync } from "node:child_process";
import {
  mkdtempSync,
  mkdirSync,
  writeFileSync,
  rmSync,
  chmodSync,
  unlinkSync,
  symlinkSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import * as admission from "./promotion.mjs";

test("selection receipt refresh retains the frozen parent after merge", () => {
  const before = "a".repeat(40);
  const after = "b".repeat(40);
  assert.equal(typeof admission.selectionProduction, "function");
  assert.equal(admission.selectionProduction(before, before), before);
  assert.equal(
    admission.selectionProduction(after, before, before, after),
    before,
  );
  assert.throws(
    () => admission.selectionProduction("c".repeat(40), before, before, after),
    /Production advanced/,
  );
  assert.throws(
    () => admission.selectionProduction(after, before),
    /Production advanced/,
  );
  assert.throws(
    () => admission.selectionProduction(after, before, after, after),
    /Frozen production parent/,
  );
});

function fixture(run) {
  const cwd = mkdtempSync(join(tmpdir(), "burrow-site-inputs-"));
  const git = (...args) =>
    execFileSync("git", args, { cwd, encoding: "utf8" }).trim();
  const put = (path, value) => {
    mkdirSync(dirname(join(cwd, path)), { recursive: true });
    writeFileSync(join(cwd, path), value);
  };
  const commit = () => {
    git("add", ".");
    git("commit", "-qm", "fixture");
    return git("rev-parse", "HEAD");
  };
  try {
    git("init", "-q");
    git("config", "user.email", "fixture@example.invalid");
    git("config", "user.name", "Fixture");
    put("package.json", JSON.stringify({ private: true, version: "0.3.24" }));
    put(".release-please-manifest.json", JSON.stringify({ ".": "0.3.24" }));
    put("scripts/install.sh", "verified installer\n");
    put("scripts/ci/current-ref.mjs", "verified writer fence\n");
    put("scripts/ci/github-json.mjs", "verified provider reader\n");
    put(
      "packages/burrowctl/package.json",
      '{"version":"0.3.23","optionalDependencies":{"native":"0.3.23"}}',
    );
    put("burrowd/main.go", "package main\n");
    put("README.md", "archive content\n");
    const baseline = commit();
    run({ cwd, git, put, commit, baseline });
  } finally {
    rmSync(cwd, { recursive: true, force: true });
  }
}

test("successive source-bound site releases need no tag or tooling PR exception", () => {
  assert.equal(typeof admission.verifySiteOnlyInputs, "function");
  fixture(({ cwd, put, commit, baseline }) => {
    let production = baseline;
    for (const version of ["0.3.25", "0.3.26"]) {
      put("packages/website/src/app/page.tsx", `site ${version}`);
      put("documentation/deploy.md", `docs ${version}`);
      put("scripts/release/promotion.mjs", `tooling ${version}`);
      put("scripts/ci/selection.mjs", `selector ${version}`);
      put("scripts/ci/selection.test.mjs", `selector tests ${version}`);
      put("package.json", JSON.stringify({ private: true, version }));
      put(".release-please-manifest.json", JSON.stringify({ ".": version }));
      put("CHANGELOG.md", `notes ${version}`);
      const source = commit();
      const proof = admission.verifySiteOnlyInputs({
        baseline,
        production,
        source,
        tag: `v${version}`,
        cwd,
      });
      assert.equal(proof.baseline, baseline);
      assert.equal(proof.production, production);
      assert.equal(proof.source, source);
      assert.match(proof.inputs, /^[a-f0-9]{64}$/);
      production = source;
    }
  });
});

test("real frozen/merge graphs renew identical proofs only at the selected merge tip", () => {
  fixture(({ cwd, git, put, commit, baseline }) => {
    let production = baseline;
    for (const version of ["0.3.25", "0.3.26"]) {
      put("packages/website/page.tsx", version);
      put("package.json", JSON.stringify({ private: true, version }));
      put(".release-please-manifest.json", JSON.stringify({ ".": version }));
      const source = commit();
      const tree = git("rev-parse", `${source}^{tree}`);
      const freeze = git(
        "commit-tree",
        tree,
        "-p",
        production,
        "-p",
        source,
        "-m",
        "freeze",
      );
      const merged = git(
        "commit-tree",
        tree,
        "-p",
        production,
        "-p",
        freeze,
        "-m",
        "merge",
      );
      const proof = admission.verifySiteOnlyInputs({
        baseline,
        production,
        source,
        tag: `v${version}`,
        cwd,
      });
      const retained = admission.selectionProduction(
        merged,
        git("rev-parse", `${freeze}^1`),
        git("rev-parse", `${merged}^1`),
        merged,
      );
      assert.deepEqual(
        admission.verifySiteOnlyInputs({
          baseline,
          production: retained,
          source,
          tag: `v${version}`,
          cwd,
        }),
        proof,
      );
      const laterSameTree = git(
        "commit-tree",
        tree,
        "-p",
        merged,
        "-m",
        "later same-tree production",
      );
      assert.throws(
        () =>
          admission.selectionProduction(
            laterSameTree,
            production,
            production,
            merged,
          ),
        /Production advanced/,
      );
      production = merged;
    }
  });
});

for (const path of [
  "scripts/install.sh",
  "packages/burrowctl/package.json",
  "packages/burrowctl-linux-x64/package.json",
  "burrowd/main.go",
  "burrowctl/version.go",
  "protocol/go.mod",
  "pnpm-lock.yaml",
  "go.work",
  "Dockerfile",
  ".dockerignore",
  ".goreleaser.yml",
  "infra/production/main.tf",
  "scripts/infra/apply.sh",
  ".github/workflows/deploy.yml",
  "README.md",
  "LICENSE",
  "scripts/release/src/index.ts",
  "scripts/ci/current-ref.mjs",
  "scripts/ci/github-json.mjs",
  "scripts/ci/new-writer-helper.mjs",
  "unknown-input",
  "packages/unknown/package.json",
])
  test(`site-only refuses changed delivery input ${path}`, () => {
    fixture(({ cwd, put, commit, baseline }) => {
      put(path, "changed\n");
      const source = commit();
      assert.throws(
        () =>
          admission.verifySiteOnlyInputs({
            baseline,
            production: baseline,
            source,
            tag: "v0.3.24",
            cwd,
          }),
        new RegExp(path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")),
      );
    });
  });

test("installer drift on production also refuses a later candidate", () => {
  fixture(({ cwd, git, put, commit, baseline }) => {
    put("scripts/install.sh", "changed");
    const production = commit();
    git("checkout", "-q", baseline);
    put("packages/website/page.tsx", "site");
    const source = commit();
    assert.throws(
      () =>
        admission.verifySiteOnlyInputs({
          baseline,
          production,
          source,
          tag: "v0.3.24",
          cwd,
        }),
      /scripts\/install.sh/,
    );
  });
});

test("root scripts and malformed release ledgers are not release-only metadata", () => {
  fixture(({ cwd, put, commit, baseline }) => {
    put(
      "package.json",
      JSON.stringify({
        private: true,
        version: "0.3.24",
        scripts: { install: "changed" },
      }),
    );
    let source = commit();
    assert.throws(
      () =>
        admission.verifySiteOnlyInputs({
          baseline,
          production: baseline,
          source,
          tag: "v0.3.24",
          cwd,
        }),
      /package.json/,
    );
    put("package.json", JSON.stringify({ private: true, version: "0.3.24" }));
    put(".release-please-manifest.json", '{".":"0.3.24","native":"0.3.25"}');
    source = commit();
    assert.throws(
      () =>
        admission.verifySiteOnlyInputs({
          baseline,
          production: baseline,
          source,
          tag: "v0.3.24",
          cwd,
        }),
      /manifest/,
    );
  });
});

for (const kind of ["delete", "rename", "mode", "symlink"]) {
  test(`retained input ${kind} cannot evade content comparison`, () => {
    fixture(({ cwd, git, commit, baseline }) => {
      const path = join(cwd, "scripts/install.sh");
      if (kind === "delete") unlinkSync(path);
      if (kind === "rename") {
        mkdirSync(join(cwd, "documentation"));
        git("mv", "scripts/install.sh", "documentation/install.sh");
      }
      if (kind === "mode") chmodSync(path, 0o755);
      if (kind === "symlink") {
        unlinkSync(path);
        symlinkSync("../README.md", path);
      }
      const source = commit();
      assert.throws(
        () =>
          admission.verifySiteOnlyInputs({
            baseline,
            production: baseline,
            source,
            tag: "v0.3.24",
            cwd,
          }),
        /scripts\/install.sh/,
      );
    });
  });
}
