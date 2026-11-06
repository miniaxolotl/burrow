import { createHash } from "node:crypto";
import { expect, test } from "./fixtures";

test("Burrow uses approved theme-aware identity without kit website graphics", async ({
  page,
  request,
}) => {
  await page.goto("/");
  for (const theme of ["light", "dark"]) {
    await expect(
      page.locator(
        `link[rel="icon"][media="(prefers-color-scheme: ${theme})"]`,
      ),
    ).toHaveCount(2);
  }
  const response = await request.get("/marketing-kit.json");
  expect(response.ok()).toBe(true);
  const manifest = await response.json();
  expect(manifest.brand).toBe("burrow");
  expect(manifest.websiteGraphicsAllowed).toBe(false);
  for (const asset of manifest.assets) {
    expect(asset.source).not.toContain("website-graphics");
    const url = asset.file.startsWith("public/")
      ? asset.file.slice("public/".length)
      : asset.file.split("/").at(-1);
    const file = await request.get(`/${url}`);
    expect(file.ok(), url).toBe(true);
    expect(
      createHash("sha256")
        .update(await file.body())
        .digest("hex"),
      url,
    ).toBe(asset.sha256);
  }
  const icon = await request.get("/icon.svg");
  expect(await icon.text()).toContain("prefers-color-scheme");
  await expect(
    page.locator('img[src*="/images/marketing"], img[src*="website-graphics"]'),
  ).toHaveCount(0);
  await expect(
    page.getByLabel("Illustrative frontend and API ports"),
  ).toContainText("localhost:3000");
});
