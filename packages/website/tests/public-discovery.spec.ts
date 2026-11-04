import { expect, test } from "@playwright/test";

test.use({ javaScriptEnabled: false });

test("public AI guide is readable, linked and describes real CLI setup", async ({
  page,
  request,
}) => {
  const response = await request.get("/llms.txt");
  expect(response.status()).toBe(200);
  expect(response.headers()["content-type"]).toContain("text/plain");
  const text = await response.text();
  expect(text).toMatch(/^# Burrow/m);
  expect(text).toContain("burrowctl tunnel 3000");
  expect(text).toContain("macOS and Linux");
  expect(text).toContain("not an access-control boundary");
  expect(text).not.toMatch(/<html|api[_-]?key|secret|token=/i);
  await page.goto("/");
  await expect(
    page.locator('link[rel="alternate"][type="text/plain"]'),
  ).toHaveAttribute("href", "/llms.txt");
});
