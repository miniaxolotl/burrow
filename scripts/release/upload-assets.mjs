import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const tag = process.argv[2];
if (
  !/^v\d+\.\d+\.\d+$/.test(tag) ||
  process.env.GITHUB_REF !== "refs/heads/production"
)
  throw new Error("Production release tag required");
const repository = "miniaxolotl/burrow";
const release = JSON.parse(
  execFileSync("gh", ["api", `repos/${repository}/releases/tags/${tag}`], {
    encoding: "utf8",
  }),
);
const files = readdirSync("dist").filter(
  (name) => name.endsWith(".tar.gz") || name === "checksums.txt",
);
if (files.length !== 9)
  throw new Error("Expected eight binary archives and checksums");
const pending = [];
for (const name of files) {
  const artifact = release.assets.find((asset) => asset.name === name);
  const bytes = readFileSync(join("dist", name));
  if (artifact) {
    const existing = execFileSync(
      "gh",
      [
        "api",
        `repos/${repository}/releases/assets/${artifact.id}`,
        "-H",
        "Accept: application/octet-stream",
      ],
      { maxBuffer: 100 * 1024 * 1024 },
    );
    const hash = (value) => createHash("sha256").update(value).digest("hex");
    if (hash(existing) !== hash(bytes))
      throw new Error(
        `Existing ${name} differs; never overwrite released artifacts`,
      );
  } else pending.push(join("dist", name));
}
if (!process.argv.includes("--dry-run") && pending.length)
  execFileSync(
    "gh",
    ["release", "upload", tag, ...pending, "--repo", repository],
    { stdio: "inherit" },
  );
