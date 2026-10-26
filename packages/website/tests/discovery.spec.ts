import { expect, test } from "./fixtures";

test.describe("public reference documents", () => {
  test.use({ javaScriptEnabled: false });
  for (const [route, title] of [
    ["privacy", "Privacy policy"],
    ["terms", "Terms and conditions"],
  ]) {
    test(`legal ${route} shares its own title and description`, async ({
      page,
    }) => {
      await page.goto(`/${route}`);
      const description = await page
        .locator('meta[name="description"]')
        .getAttribute("content");
      for (const selector of [
        'meta[property="og:title"]',
        'meta[name="twitter:title"]',
      ]) {
        await expect(page.locator(selector)).toHaveAttribute(
          "content",
          `${title} | Burrow`,
        );
      }
      for (const selector of [
        'meta[property="og:description"]',
        'meta[name="twitter:description"]',
      ]) {
        await expect(page.locator(selector)).toHaveAttribute(
          "content",
          description!,
        );
      }
    });
  }
  for (const [slug, heading, content] of [
    ["client", "burrowctl", "burrowctl tunnel 3000 8080"],
    ["server", "burrowd", "X-Tunnel-Token"],
    ["workflows", "Webhooks and multiple local services", "204 No Content"],
  ]) {
    test(`${slug} has readable canonical documentation`, async ({ page }) => {
      expect((await page.goto(`/docs/${slug}`))?.status()).toBe(200);
      await expect(page.locator("main h1")).toHaveCount(1);
      await expect(page.locator("main h1")).toContainText(heading);
      await expect(page.locator("article")).toContainText(content);
      await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
        "href",
        `http://localhost:3000/docs/${slug}`,
      );
      await expect(page.locator('meta[property="og:url"]')).toHaveAttribute(
        "content",
        `http://localhost:3000/docs/${slug}`,
      );
      // Local builds intentionally publish no sitemap entries. Verify production
      // coverage separately with SITE_URL set before the production smoke build.
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
        "content",
        "noindex, nofollow",
      );
      expect(
        await (await page.request.get("/sitemap.xml")).text(),
      ).not.toContain("<loc>");
    });
  }

  test("hub and Markdown links resolve without executing JavaScript", async ({
    page,
  }) => {
    await page.goto("/");
    await page
      .getByRole("contentinfo")
      .getByRole("link", { name: "Documentation", exact: true })
      .click();
    await expect(page).toHaveURL(/\/docs$/);
    await page
      .getByRole("article")
      .getByRole("link", { name: "burrowctl", exact: true })
      .click();
    await expect(page.locator("#automatic-ownership")).toHaveCount(1);
    await expect(page.locator('article a[href="/docs/server"]')).toBeVisible();
    await expect(
      page.locator(
        'article a[href="https://github.com/miniaxolotl/burrow/blob/production/documentation/deploy.md"]',
      ),
    ).toBeVisible();
    await page.goto("/docs/workflows");
    await expect(page.locator("article")).toContainText("not the application");
    await expect(page.locator("article")).toContainText(
      "Keep the private owner configuration",
    );
  });

  test("only intentionally published documents resolve", async ({
    request,
  }) => {
    for (const path of ["/docs/secrets", "/docs/deploy", "/docs/missing"]) {
      expect((await request.get(path)).status()).toBe(404);
    }
  });
});

test("reference code and tables do not widen the viewport", async ({
  page,
}) => {
  expect((await page.goto("/docs/client"))?.status()).toBe(200);
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 950 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  }
});
