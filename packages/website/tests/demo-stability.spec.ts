import { expect, test } from "@playwright/test";

for (const width of [1440, 768, 390, 320]) {
  test(`examples keep surrounding content still at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto("/");
    await page.evaluate(() => document.fonts.ready);
    for (const [buttons, shell] of [
      [".tunnel-scenarios button", ".tunnel-landscape"],
      [".manager-tunnel", ".manager-window"],
      [".manager-tabs button", ".manager-window"],
    ]) {
      const frame = page.locator(shell);
      const height = await frame.evaluate(
        (element) => element.getBoundingClientRect().height,
      );
      for (const button of await page.locator(buttons).all()) {
        await button.click();
        await expect
          .poll(async () =>
            Math.abs(
              (await frame.evaluate(
                (element) => element.getBoundingClientRect().height,
              )) - height,
            ),
          )
          .toBeLessThan(1);
      }
    }
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    // Audit the visible lower-page instructions, not only the initial hero.
    await page.locator("#install").scrollIntoViewIfNeeded();
    await expect(page.locator("#install")).toHaveCSS(
      "background-color",
      "rgba(0, 0, 0, 0)",
    );
    await expect(page.locator("#install")).toHaveCSS(
      "color",
      "rgb(37, 62, 49)",
    );
    await expect(
      page.locator('.manager-tabs button[aria-pressed="false"]'),
    ).toHaveCSS("color", "rgb(37, 62, 49)");
  });
}
