import "server-only";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { documentation } from "@/content/documentation";

export async function loadDocumentation(slug: string) {
  const document = documentation.find((item) => item.slug === slug);
  if (!document) return null;
  // Only committed allowlisted content is read, during static generation from
  // the packages/website working directory. Request input is never a file path.
  const markdown = await readFile(
    resolve(process.cwd(), "../../documentation", document.source),
    "utf8",
  );
  return { ...document, markdown: markdown.replace(/^# [^\r\n]+\r?\n/, "") };
}
