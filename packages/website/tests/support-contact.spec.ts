import { expect, test } from "@playwright/test";

test("current public contacts use the Burrow support mailbox", async ({
  page,
}) => {
  for (const path of ["/", "/privacy", "/terms", "/missing-help-page"]) {
    await page.goto(path);
    await expect(
      page.locator('a[href^="mailto:"][href*="@mawa.dev"]'),
    ).toHaveCount(0);
  }
  await page.goto("/privacy");
  await expect(
    page.locator('main a[href="mailto:support@useburrow.app"]').first(),
  ).toBeVisible();
});
