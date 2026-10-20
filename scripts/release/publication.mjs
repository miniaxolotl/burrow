import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
} from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { setTimeout } from "node:timers/promises";

export async function publicationPlan(artifacts, lookup) {
  const pending = [];
  for (const artifact of artifacts) {
    const existing = await lookup(artifact.name);
    if (existing && existing !== artifact.integrity)
      throw new Error(
        `${artifact.name} already exists with different contents; release a new version`,
      );
    if (!existing) pending.push(artifact);
  }
  return pending;
}

export async function publishPlan(plan, publish) {
  for (const artifact of plan) await publish(artifact);
}

export function registryIntegrity(name) {
  try {
    const integrity = execFileSync(
      "npm",
      ["view", name, "dist.integrity", "--prefer-online"],
      {
        encoding: "utf8",
        stdio: ["ignore", "pipe", "pipe"],
      },
    ).trim();
    if (!integrity) throw new Error(`Missing registry integrity for ${name}`);
    return integrity;
  } catch (error) {
    if (/E404/.test(String(error.stderr))) return null;
    throw error;
  }
}

export async function waitForPublication(
  artifact,
  { lookup = registryIntegrity, wait = setTimeout, attempts = 60 } = {},
) {
  for (let attempt = 0; attempt < attempts; attempt++) {
    const integrity = await lookup(artifact.name);
    if (integrity === artifact.integrity) return;
    if (integrity)
      throw new Error(
        `Publication readback differs for ${artifact.name}; inspect before retrying`,
      );
    if (attempt + 1 < attempts) await wait(5000);
  }
  throw new Error(
    `Publication not visible for ${artifact.name}; inspect and retry without overwriting`,
  );
}

export async function publishPackages(directories, dryRun) {
  const base = process.env.CI ? tmpdir() : "/tmp/opencode";
  mkdirSync(base, { recursive: true });
  const temporary = mkdtempSync(join(base, "burrow-publish-"));
  try {
    const artifacts = directories.map((directory, i) => {
      const pkg = JSON.parse(
        readFileSync(join(directory, "package.json"), "utf8"),
      );
      const destination = join(temporary, String(i));
      mkdirSync(destination);
      execFileSync("pnpm", ["pack", "--pack-destination", destination], {
        cwd: directory,
        stdio: ["ignore", "pipe", "pipe"],
      });
      const files = readdirSync(destination).filter((name) =>
        name.endsWith(".tgz"),
      );
      if (files.length !== 1)
        throw new Error(`Expected one archive for ${pkg.name}`);
      const archive = join(destination, files[0]);
      return {
        name: `${pkg.name}@${pkg.version}`,
        archive,
        integrity: `sha512-${createHash("sha512").update(readFileSync(archive)).digest("base64")}`,
      };
    });
    const plan = await publicationPlan(artifacts, registryIntegrity);
    await publishPlan(plan, async (pkg) => {
      if (dryRun) console.log(`Would publish ${pkg.name}`);
      else {
        execFileSync(
          "npm",
          [
            "publish",
            pkg.archive,
            "--access",
            "public",
            "--ignore-scripts",
            "--provenance",
          ],
          { stdio: "inherit" },
        );
        await waitForPublication(pkg);
      }
    });
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
}
