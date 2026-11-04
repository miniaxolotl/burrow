import { expect, test } from "@playwright/test";

test("example scenarios show the matching local port and incoming request without opening a tunnel", async ({
  page,
}) => {
  const outbound: string[] = [];
  page.on("request", (request) => {
    if (new URL(request.url()).hostname.endsWith("brmbl.dev"))
      outbound.push(request.url());
  });
  await page.goto("/");
  const example = page.getByRole("figure", {
    name: /A local development server connects/,
  });
  const output = example.locator("#tunnel-example-output");
  await example
    .getByRole("button", { name: "Test a webhook", exact: true })
    .click();
  await expect(output.locator(".tunnel-command")).toContainText(
    "burrowctl tunnel 8080",
  );
  await expect(
    output.getByText("POST /webhooks/order", { exact: true }),
  ).toBeVisible();
  await expect(output.getByText("202 Accepted", { exact: true })).toBeVisible();
  await example
    .getByRole("button", { name: "Preview an API", exact: true })
    .click();
  await expect(output.locator(".tunnel-command")).toContainText(
    "burrowctl tunnel 4000",
  );
  await expect(
    output.getByText("GET /api/projects", { exact: true }),
  ).toBeVisible();
  expect(outbound).toEqual([]);
});

test("the tunnel example can be replayed by keyboard with reduced motion", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  const example = page.getByRole("figure", {
    name: /A local development server connects/,
  });
  const replay = example.getByRole("button", { name: "Replay example" });
  const output = example.locator("#tunnel-example-output");
  await replay.focus();
  await page.keyboard.press("Enter");
  await expect(replay).toBeFocused();
  await expect(output.getByText("Tunnel connected")).toBeVisible();
  await expect(output.getByText("200 OK", { exact: true })).toBeVisible();
  await expect(
    example.getByText("Illustrated preview with example URL and request"),
  ).toBeVisible();
  await expect(output.locator(".tunnel-command")).toHaveCSS(
    "animation-name",
    "none",
  );
  await expect(example.locator(".tunnel-scene")).toHaveCount(0);
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
