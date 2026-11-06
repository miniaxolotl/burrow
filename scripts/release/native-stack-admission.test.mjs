import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { selectEvent } from "../ci/selection.mjs";
import { readFileSync } from "node:fs";

const repository = "miniaxolotl/burrow";
const pr = {
  number: 88,
  state: "open",
  base: {
    ref: "docs/tunnels-development-plan",
    sha: "a".repeat(40),
    repo: { full_name: repository },
  },
  head: {
    ref: "feat/tunnel-workflows",
    sha: "b".repeat(40),
    repo: { full_name: repository },
  },
  stack: {
    id: 123,
    number: 89,
    position: 2,
    size: 4,
    base: { ref: "development", sha: "c".repeat(40) },
  },
};

function admit(t, eventPr = pr, fetchedPr = pr) {
  const directory = mkdtempSync(
    join(
      existsSync("/tmp/opencode") ? "/tmp/opencode" : tmpdir(),
      "burrow-stack-",
    ),
  );
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const event = join(directory, "event.json");
  const output = join(directory, "output");
  writeFileSync(event, JSON.stringify({ pull_request: eventPr }));
  writeFileSync(join(directory, "fresh.json"), JSON.stringify(fetchedPr));
  // Only the authenticated GitHub read is substituted. Run the real admission
  // entrypoint; any unexpected provider call (including a write) fails closed.
  writeFileSync(
    join(directory, "gh"),
    `#!${process.execPath}
const fs = require('node:fs');
const args = process.argv.slice(2);
if (JSON.stringify(args) !== JSON.stringify(['api','--hostname','github.com','--method','GET','--include','repos/${repository}/pulls/88'])) process.exit(99);
process.stdout.write('HTTP/2.0 200 OK\\ncontent-type: application/json\\n\\n' + fs.readFileSync(${JSON.stringify(join(directory, "fresh.json"))}, 'utf8'));
`,
    { mode: 0o700 },
  );
  const result = spawnSync(
    process.execPath,
    ["scripts/release/promotion.mjs", "admit"],
    {
      encoding: "utf8",
      timeout: 10_000,
      env: {
        ...process.env,
        PATH: directory + ":" + process.env.PATH,
        GITHUB_REPOSITORY: repository,
        GITHUB_EVENT_NAME: "pull_request",
        GITHUB_EVENT_PATH: event,
        GITHUB_OUTPUT: output,
        GITHUB_REF_NAME: "88/merge",
      },
    },
  );
  return {
    ...result,
    promotionOutput: existsSync(output) ? readFileSync(output, "utf8") : "",
  };
}

test("native development stacks pass source admission without promotion reuse", (t) => {
  const result = admit(t);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.promotionOutput, "");
  const policy = JSON.parse(readFileSync("scripts/ci/job-policy.json", "utf8"));
  const selected = selectEvent(policy, "pull_request", pr.base.ref, false, [
    "README.md",
  ]);
  assert.ok(
    Object.values(selected).every(Boolean),
    "stack CI must remain full",
  );
});

test("stack admission rejects event claims absent from fresh GitHub metadata", (t) => {
  assert.notEqual(admit(t, pr, { ...pr, stack: null }).status, 0);
});

test("stack admission fails closed on identity drift, foreign scope and invalid roots", (t) => {
  for (const change of [
    { number: 89 },
    { state: "closed" },
    { head: { ...pr.head, sha: "d".repeat(40) } },
    { head: { ...pr.head, ref: "feat/other" } },
    { head: { ...pr.head, repo: { full_name: "fork/burrow" } } },
    { base: { ...pr.base, sha: "d".repeat(40) } },
    { base: { ...pr.base, ref: "feat/other" } },
    { base: { ...pr.base, repo: { full_name: "fork/burrow" } } },
    { stack: { ...pr.stack, base: { ...pr.stack.base, ref: "production" } } },
    { stack: { ...pr.stack, base: { ...pr.stack.base, sha: "invalid" } } },
    { stack: { ...pr.stack, id: 0 } },
    { stack: { ...pr.stack, position: 0 } },
    { stack: { ...pr.stack, position: 5 } },
  ]) {
    assert.notEqual(
      admit(t, pr, { ...pr, ...change }).status,
      0,
      JSON.stringify(change),
    );
  }
});

test("a stack field cannot bypass production promotion admission", (t) => {
  const production = { ...pr, base: { ...pr.base, ref: "production" } };
  const result = admit(t, production, production);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /frozen release ref/i);
  assert.equal(result.promotionOutput, "");
});
