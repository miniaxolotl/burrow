import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { test } from "node:test";

for (const supersede of [false, true])
  test(`each infrastructure write is fenced; superseded=${supersede}`, () => {
    const directory = mkdtempSync(join(tmpdir(), "burrow-fence-"));
    const log = join(directory, "calls");
    const sha = "a".repeat(40);
    const executable = (name, source) =>
      writeFileSync(join(directory, name), source, { mode: 0o700 });
    executable(
      "tofu",
      `#!/bin/sh\necho "$*" >> "$FENCE_LOG"\ncase "$*" in *"output -raw url"*) echo https://fixture.invalid;; esac\n`,
    );
    executable("python3", "#!/bin/sh\nexit 0\n");
    executable("curl", "#!/bin/sh\nexit 0\n");
    executable(
      "gh",
      `#!/usr/bin/env node
import {appendFileSync,readFileSync} from 'node:fs';
const log=process.env.FENCE_LOG;
const previous=readFileSync(log,'utf8').split('fence').length-1;
appendFileSync(log,'fence\\n');
process.stdout.write('HTTP/2.0 200 OK\\nContent-Type: application/json\\n\\n'+JSON.stringify({object:{sha:process.env.SUPERSEDE==='true'&&previous>0?'${"b".repeat(40)}':'${sha}'}}));
`,
    );
    try {
      const result = spawnSync(
        "bash",
        ["scripts/infra/apply.sh", "production"],
        {
          encoding: "utf8",
          timeout: 30_000,
          env: {
            ...process.env,
            PATH: `${directory}:${process.env.PATH}`,
            FENCE_LOG: log,
            SUPERSEDE: String(supersede),
            GITHUB_ACTIONS: "true",
            GITHUB_REPOSITORY: "miniaxolotl/burrow",
            GITHUB_REF: "refs/heads/production",
            GITHUB_SHA: sha,
            GITHUB_REF_PROTECTED: "true",
            GH_TOKEN: "fixture",
            PG_CONN_STR: "fixture",
            TF_ENCRYPTION: "fixture",
            DIGITALOCEAN_ACCESS_TOKEN: "fixture",
            CLOUDFLARE_API_TOKEN: "fixture",
            UPSTASH_EMAIL: "fixture",
            UPSTASH_API_KEY: "fixture",
            TF_VAR_image_digest: `sha256:${"c".repeat(64)}`,
          },
        },
      );
      const calls = readFileSync(log, "utf8").trim().split("\n");
      assert.equal(result.status, supersede ? 1 : 0, result.stderr);
      const applies = calls.filter((line) => line.includes(" apply "));
      assert.equal(applies.length, supersede ? 1 : 2);
      for (let i = 0; i < calls.length; i++)
        if (calls[i].includes(" apply ")) assert.equal(calls[i - 1], "fence");
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
