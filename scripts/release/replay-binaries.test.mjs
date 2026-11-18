import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const script = fileURLToPath(new URL("./replay-binaries.sh", import.meta.url));

function fixture(t) {
  const cwd = mkdtempSync(join(tmpdir(), "burrow-replay-test-"));
  t.after(() => rmSync(cwd, { recursive: true, force: true }));
  const git = (...args) => execFileSync("git", args, { cwd });
  const write = (path, contents) => writeFileSync(join(cwd, path), contents);
  const commit = () => {
    git("add", ".");
    git(
      "-c",
      "user.name=Fixture",
      "-c",
      "user.email=test@example.org",
      "-c",
      "commit.gpgsign=false",
      "commit",
      "-qm",
      "fixture",
    );
  };
  git("init", "-q");
  write("go.mod", "module example.org/replay\n\ngo 1.20\n");
  for (const id of ["burrowd", "burrowctl"]) {
    mkdirSync(join(cwd, id));
    write(`${id}/main.go`, "package main\nfunc main() { println(Version) }\n");
    write(
      `${id}/version.go`,
      'package main\nvar Version = "1.0.0" // x-release-please-version\n',
    );
  }
  const replay = (target = "linux/amd64", env = {}, base = "HEAD^") =>
    spawnSync("bash", [script, base, "HEAD", target], {
      cwd,
      encoding: "utf8",
      env: { ...process.env, ...env },
    });
  return { git, write, commit, replay };
}

test("compare committed payloads independently of stamp, test and working-tree edits", (t) => {
  const { write, commit, replay, git } = fixture(t);
  commit();
  write(
    "burrowd/main.go",
    'package main\nfunc main() { println("new service", Version) }\n',
  );
  write(
    "burrowctl/version.go",
    'package main\nvar Version = "1.0.1" // x-release-please-version\n',
  );
  write("burrowctl/main_test.go", "package main\n");
  commit();
  write("burrowctl/main.go", "not committed and must not be compiled");
  const before = git("worktree", "list", "--porcelain").toString();
  const result = replay();
  assert.equal(result.status, 0, result.stderr);
  assert.match(
    result.stdout,
    /^burrowd\s+different\s+[a-f0-9]{64}\s+[a-f0-9]{64}$/m,
  );
  assert.match(result.stdout, /^burrowctl\s+identical\s+([a-f0-9]{64})\s+\1$/m);
  assert.match(result.stdout, /burrowctl\/main_test\.go/);
  assert.match(result.stdout, /report-only/i);
  assert.equal(git("worktree", "list", "--porcelain").toString(), before);
  assert.notEqual(replay("linux/amd64", {}, "missing").status, 0);
  commit(); // Invalid committed Go source must fail without leaving checkouts behind.
  const beforeFailure = git("worktree", "list", "--porcelain").toString();
  const failed = replay();
  assert.notEqual(failed.status, 0);
  assert.doesNotMatch(failed.stdout, /Report-only binary comparison/);
  assert.equal(
    git("worktree", "list", "--porcelain").toString(),
    beforeFailure,
  );
});

test("a tracked export-ignore source edit changes the checkout-built server", (t) => {
  const { write, commit, replay } = fixture(t);
  write(".gitattributes", "burrowd/banner.go export-ignore\n");
  for (const banner of ["before", "after"]) {
    write(
      "burrowd/banner.go",
      `package main\nfunc init() { println("${banner}") }\n`,
    );
    commit();
  }
  const result = replay();
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /^burrowd\s+different\b/m);
  assert.match(result.stdout, /^burrowctl\s+identical\b/m);
});

for (const [arch, setting, release, caller, higher] of [
  ["amd64", "GOAMD64", "v1", "v3", "amd64.v3"],
  ["arm64", "GOARM64", "v8.0", "v8.2", "arm64.v8.2"],
]) {
  test(`${arch} replay ignores caller tuning and reports the release CPU level`, (t) => {
    const { write, commit, replay } = fixture(t);
    for (const value of ["before", "after"]) {
      write(
        "burrowd/baseline.go",
        `//go:build !${higher}\n\npackage main\nfunc init() { println("${value}") }\n`,
      );
      commit();
    }
    const run = (level) => replay(`linux/${arch}`, { [setting]: level });
    const baseline = run(release);
    const tuned = run(caller);
    assert.equal(baseline.status, 0, baseline.stderr);
    assert.equal(tuned.status, 0, tuned.stderr);
    assert.match(baseline.stdout, /^burrowd\s+different\b/m);
    assert.equal(tuned.stdout, baseline.stdout);
    assert.ok(tuned.stdout.includes(`${setting}=${release}`));
  });
}
