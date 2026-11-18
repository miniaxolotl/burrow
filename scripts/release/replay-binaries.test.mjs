import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const script = fileURLToPath(new URL("./replay-binaries.sh", import.meta.url));

test("compare committed payloads independently of stamp, test and working-tree edits", (t) => {
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
  const replay = (...args) =>
    spawnSync("bash", [script, ...args], { cwd, encoding: "utf8" });
  const result = replay("HEAD^", "HEAD", "linux/amd64");
  assert.equal(result.status, 0, result.stderr);
  assert.match(
    result.stdout,
    /^burrowd\s+different\s+[a-f0-9]{64}\s+[a-f0-9]{64}$/m,
  );
  assert.match(result.stdout, /^burrowctl\s+identical\s+([a-f0-9]{64})\s+\1$/m);
  assert.match(result.stdout, /burrowctl\/main_test\.go/);
  assert.match(result.stdout, /report-only/i);
  assert.notEqual(replay("missing", "HEAD", "linux/amd64").status, 0);
});
