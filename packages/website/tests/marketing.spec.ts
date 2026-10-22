import { expect, test } from "@playwright/test";

test("the tunnel example can be replayed by keyboard with reduced motion", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  const example = page.getByRole("figure", {
    name: /A local development server connects/,
  });
  const replay = example.getByRole("button", { name: "Replay example" });
  await replay.focus();
  await page.keyboard.press("Enter");
  await expect(replay).toBeFocused();
  await expect(example.getByText("Tunnel connected")).toBeVisible();
  await expect(example.getByText("200 OK", { exact: true })).toBeVisible();
  await expect(
    example.getByText("Illustrated preview with example URL and request"),
  ).toBeVisible();
  await expect(example.locator(".tunnel-command")).toHaveCSS(
    "animation-name",
    "none",
  );
  await expect(example.locator(".tunnel-route path")).toHaveCSS(
    "animation-name",
    "none",
  );
});

test("the manager keeps the selected tunnel when opening its example requests", async ({
  page,
}) => {
  await page.goto("/");
  const manager = page.locator("#how-it-works");
  const selection = manager.getByRole("button", {
    name: "Select webhooks tunnel on port 8080",
  });
  await selection.focus();
  await page.keyboard.press("Enter");
  await expect(selection).toBeFocused();
  await expect(selection).toHaveAttribute("aria-pressed", "true");
  await manager
    .getByRole("button", { name: "View requests", exact: true })
    .click();
  await expect(
    manager.getByRole("table", {
      name: /Example requests for the webhooks tunnel on port 8080/,
    }),
  ).toBeVisible();
  await expect(
    manager.getByRole("cell", { name: "/webhooks/order", exact: true }),
  ).toBeVisible();
  await manager
    .getByRole("button", { name: "Back to tunnels", exact: true })
    .click();
  await expect(selection).toHaveAttribute("aria-pressed", "true");
});
