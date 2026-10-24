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

for (const route of ["support", "contact"]) {
  test(`${route} is public with correct metadata, navigation and mobile layout`, async ({
    page,
    request,
  }) => {
    const origin = new URL(
      process.env.SITE_URL ||
        (process.env.VERCEL_PROJECT_PRODUCTION_URL &&
          `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`) ||
        "http://localhost:3000",
    ).origin;
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    expect((await page.goto(`/${route}`))?.status()).toBe(200);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(
      route === "support" ? "Support for Burrow." : "Contact Burrow.",
    );
    await expect(
      page.locator('main a[href="mailto:support@useburrow.app"]').first(),
    ).toBeVisible();
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
      "href",
      `${origin}/${route}`,
    );
    await expect(page).toHaveTitle(
      `${route === "support" ? "Support" : "Contact"} | Burrow`,
    );
    const footer = page.getByRole("contentinfo");
    await expect(
      footer.getByRole("link", { name: "Support", exact: true }),
    ).toHaveAttribute("href", "/support");
    await expect(
      footer.getByRole("link", { name: "Contact", exact: true }),
    ).toHaveAttribute("href", "/contact");
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 950 });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    }
    const sitemap = await (await request.get("/sitemap.xml")).text();
    if (
      !/localhost|127\.0\.0\.1/.test(origin) &&
      process.env.VERCEL_ENV !== "preview" &&
      process.env.SITE_NOINDEX !== "true"
    )
      expect(sitemap).toContain(`${origin}/${route}`);
    else expect(sitemap).not.toContain(`<loc>${origin}/${route}</loc>`);
    expect(errors).toEqual([]);
  });
}

test("help pages work without JavaScript and link to the real client docs", async ({
  page,
  browser,
  baseURL,
}) => {
  await page.goto("/support");
  await expect(
    page.getByRole("link", {
      name: "Burrow client documentation",
      exact: true,
    }),
  ).toHaveAttribute(
    "href",
    "https://github.com/miniaxolotl/burrow/blob/production/documentation/client.md",
  );
  const context = await browser.newContext({
    baseURL,
    javaScriptEnabled: false,
  });
  try {
    const plain = await context.newPage();
    for (const route of ["support", "contact"]) {
      expect((await plain.goto(`/${route}`))?.status()).toBe(200);
      await expect(
        plain.locator('main a[href="mailto:support@useburrow.app"]').first(),
      ).toBeVisible();
    }
    await plain
      .getByRole("contentinfo")
      .getByRole("link", { name: "Support", exact: true })
      .focus();
    await plain.keyboard.press("Enter");
    await expect(plain).toHaveURL(/\/support$/);
  } finally {
    await context.close();
  }
});
