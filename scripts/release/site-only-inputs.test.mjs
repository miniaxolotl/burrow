import assert from "node:assert/strict";
import { test } from "node:test";
import { execFileSync, spawnSync } from "node:child_process";
import {
  mkdtempSync,
  mkdirSync,
  writeFileSync,
  readFileSync,
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

test("an exact release push prepares one frozen PR and renews it on the original run", () => {
  fixture(({ cwd, git, put, commit, baseline }) => {
    const policy = JSON.parse(readFileSync("scripts/release/policy.json"));
    policy.siteOnly.verifiedProduction = baseline;
    const repo = policy.repository;
    put("packages/website/page.tsx", "new site");
    put("package.json", JSON.stringify({ private: true, version: "0.3.25" }));
    put(".release-please-manifest.json", JSON.stringify({ ".": "0.3.25" }));
    const sha = commit();
    const tree = git("rev-parse", "HEAD^{tree}");
    git("update-ref", "refs/remotes/origin/development", sha);
    const tag = "v0.3.25";
    const branch = `promote/production/${tag}`;
    const head = git(
      "commit-tree",
      tree,
      "-p",
      baseline,
      "-p",
      sha,
      "-m",
      "frozen",
    );
    const pr = {
      number: 17,
      state: "open",
      head: { ref: branch, sha: head, repo: { full_name: repo } },
      base: { ref: "production", sha: baseline },
      body: "<!-- release-selection:42 -->",
      user: { login: "github-actions[bot]", type: "Bot" },
    };
    const releasePr = {
      merged_at: "2026-01-01",
      merge_commit_sha: sha,
      base: { ref: "development", repo: { full_name: repo } },
      head: {
        ref: "release-please--branches--development--components--burrow",
        repo: { full_name: repo },
      },
      user: { login: "github-actions[bot]", type: "Bot" },
      labels: [{ name: "autorelease: tagged" }],
    };
    const run = {
      id: 5,
      run_attempt: 1,
      event: "push",
      status: "completed",
      conclusion: "success",
      path: ".github/workflows/ci.yml",
      head_sha: sha,
      head_repository: { full_name: repo },
    };
    const jobs = policy.ciJobs.map((name) => ({
      name,
      run_id: 5,
      run_attempt: 1,
      head_sha: sha,
      status: "completed",
      conclusion: "success",
    }));
    const listPr = `pulls?state=all&base=production&head=miniaxolotl:${branch}&per_page=100&page=1`;
    const routes = {
      [`releases/tags/${tag}`]: { tag_name: tag, draft: false },
      [`commits/${sha}/pulls?per_page=100&page=1`]: [releasePr],
      [`actions/workflows/ci.yml/runs?head_sha=${sha}&per_page=100&page=1`]: {
        workflow_runs: [run],
      },
      "actions/runs/5/attempts/1/jobs?per_page=100&page=1": { jobs },
      "git/ref/heads/production": { object: { sha: baseline } },
      [listPr]: [],
      [`git/matching-refs/heads/${branch}`]: [],
      "POST git/commits": { sha: head },
      "POST git/refs": {},
      "POST pulls": pr,
      "POST issues/17/assignees": {},
      "POST issues/17/labels": {},
    };
    for (const file of ["release/promotion.mjs", "ci/github-json.mjs"])
      put(`tools/${file}`, readFileSync(`scripts/${file}`));
    put("tools/release/policy.json", JSON.stringify(policy));
    put(
      "event.json",
      JSON.stringify({
        after: sha,
        forced: false,
        deleted: false,
        created: false,
      }),
    );
    put("writes", "");
    put(
      "gh",
      `#!${process.execPath}\nconst fs=require('node:fs'); const a=process.argv.slice(2); const key=a.find(x=>x.startsWith('repos/')).slice(${`repos/${repo}/`.length}); const write=a.includes('POST'); if(write)fs.appendFileSync('writes',JSON.stringify([key,JSON.parse(fs.readFileSync(0))])+'\\n');const v=JSON.parse(fs.readFileSync('routes'))[(write?'POST ':'')+key]; if(v===undefined)throw Error(key);console.log((write?'':'HTTP/2.0 200 OK\\n\\n')+JSON.stringify(v));`,
    );
    chmodSync(join(cwd, "gh"), 0o700);
    const invoke = (overrides = {}) => {
      put("routes", JSON.stringify(routes));
      return spawnSync(
        process.execPath,
        ["tools/release/promotion.mjs", "auto"],
        {
          cwd,
          encoding: "utf8",
          timeout: 10000,
          env: {
            ...process.env,
            PATH: `${cwd}:${process.env.PATH}`,
            GITHUB_ACTIONS: "true",
            GITHUB_REF: "refs/heads/development",
            GITHUB_REF_PROTECTED: "true",
            GITHUB_REPOSITORY: repo,
            GITHUB_SHA: sha,
            GITHUB_WORKFLOW_REF: `${repo}/.github/workflows/release.yml@refs/heads/development`,
            GITHUB_EVENT_NAME: "push",
            GITHUB_EVENT_PATH: join(cwd, "event.json"),
            GITHUB_RUN_ID: "42",
            GITHUB_OUTPUT: join(cwd, "output"),
            ...overrides,
          },
        },
      );
    };
    const writes = () =>
      readFileSync(join(cwd, "writes"), "utf8")
        .trim()
        .split("\n")
        .filter(Boolean)
        .map((s) => JSON.parse(s));
    assert.equal(
      invoke().status,
      0,
      "ordinary push skips without selecting the previous tag",
    );
    git("tag", "-a", tag, "-m", "released", sha);
    for (const overrides of [
      { GITHUB_REF_PROTECTED: "false" },
      { GITHUB_EVENT_NAME: "workflow_dispatch" },
      { GITHUB_SHA: baseline },
      { GITHUB_WORKFLOW_REF: "wrong" },
    ])
      assert.notEqual(invoke(overrides).status, 0);
    releasePr.head.ref = "feature/unrelated";
    assert.notEqual(invoke().status, 0);
    releasePr.head.ref =
      "release-please--branches--development--components--burrow";
    routes["branches/production"] = {
      name: "production",
      protected: false,
      commit: { sha: baseline },
    };
    assert.notEqual(invoke().status, 0, "production must still be protected");
    routes["branches/production"].protected = true;
    assert.deepEqual(writes(), []);
    const first = invoke();
    assert.equal(first.status, 0, first.stderr);
    assert.deepEqual(
      writes().find(([key]) => key === "git/commits")[1].parents,
      [baseline, sha],
    );
    assert.deepEqual(writes().find(([key]) => key === "git/refs")[1], {
      ref: `refs/heads/${branch}`,
      sha: head,
    });
    routes[listPr] = [pr];
    const retry = invoke();
    assert.equal(retry.status, 0, retry.stderr);
    assert.equal(writes().filter(([key]) => key === "pulls").length, 1);
    assert.equal(writes().filter(([key]) => key === "git/refs").length, 1);
    assert.equal(
      writes().filter(([key]) => key === "issues/17/labels").length,
      2,
      "retry repairs interrupted PR decoration",
    );
    const receipt = JSON.parse(readFileSync(join(cwd, "selection.json")));
    assert.equal(receipt.delivery, "site-only");
    assert.equal(receipt.sha, sha);
    assert.equal(receipt.head, head);
    assert.equal(receipt.sourceCiRun, 5);
    assert.notEqual(
      invoke({ GITHUB_RUN_ID: "43" }).status,
      0,
      "do not adopt another run's selection",
    );
  });
});

test("successive source-bound site releases need no tag or tooling PR exception", () => {
  assert.equal(typeof admission.verifySiteOnlyInputs, "function");
  fixture(({ cwd, put, commit, baseline }) => {
    let production = baseline;
    for (const version of ["0.3.25", "0.3.26"]) {
      put("packages/website/src/app/page.tsx", `site ${version}`);
      put("documentation/deploy.md", `docs ${version}`);
      put("scripts/release/promotion.mjs", `tooling ${version}`);
      put(".github/workflows/release.yml", `source release ${version}`);
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
