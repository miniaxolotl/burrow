import { test, expect } from "./fixtures";
import { sanitizeError, sanitizeSpan } from "../src/lib/diagnostics-policy";

test("Sentry's boundary removes private event, stack-context and span data", () => {
  const event = sanitizeError({
    type: undefined,
    event_id: "fixture",
    message: "PRIVATE",
    user: { email: "PRIVATE" },
    request: {
      url: "https://private.test/PRIVATE",
      headers: { Authorization: "PRIVATE" },
    },
    extra: { private: "PRIVATE" },
    exception: {
      values: [
        {
          type: "TypeError",
          value: "PRIVATE",
          stacktrace: {
            frames: [
              {
                filename:
                  "https://useburrow.app/_next/static/chunks/app.js?PRIVATE",
                lineno: 9,
                colno: 2,
                context_line: "PRIVATE",
                vars: { token: "PRIVATE" },
              },
            ],
          },
        },
      ],
    },
  });
  expect(JSON.stringify(event)).not.toContain("PRIVATE");
  expect(event.exception?.values?.[0].stacktrace?.frames?.[0]).toMatchObject({
    filename: "https://useburrow.app/_next/static/chunks/app.js",
    lineno: 9,
    colno: 2,
  });
  const span = sanitizeSpan({
    name: "https://private.test/PRIVATE",
    trace_id: "random",
    span_id: "random",
    start_timestamp: 1,
    status: "ok",
    is_segment: false,
    attributes: { "sentry.op": "http.client", "http.request.body": "PRIVATE" },
    links: [{ trace_id: "PRIVATE", span_id: "PRIVATE" }],
  });
  expect(JSON.stringify(span)).not.toContain("PRIVATE");
});

test("real SDKs emit sanitized events, errors, streamed spans, logs and metrics", async ({
  page,
  context,
  telemetry,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/?token=PRIVATE#PRIVATE", {
    referer: "https://example.com/PRIVATE?token=PRIVATE",
  });
  await page
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  await page.evaluate(() => {
    setTimeout(() => {
      throw new TypeError("PRIVATE");
    }, 0);
  });
  await expect
    .poll(
      () =>
        telemetry.events.some(
          (event) => event.event === "install_command_copied",
        ),
      { timeout: 15000 },
    )
    .toBe(true);
  await expect
    .poll(() => telemetry.sentry.some((item) => item.type === "event"), {
      timeout: 15000,
    })
    .toBe(true);
  for (const type of ["span", "log", "trace_metric"]) {
    await expect
      .poll(() => telemetry.sentry.some((item) => item.type === type), {
        timeout: 15000,
      })
      .toBe(true);
  }
  expect(
    telemetry.events.filter((event) => event.event === "$pageview"),
  ).toHaveLength(1);
  expect(
    telemetry.events.find((event) => event.event === "$pageview")?.properties,
  ).toMatchObject({
    $current_url: "https://useburrow.app/",
    $referring_domain: "example.com",
  });
  expect(JSON.stringify(telemetry.events)).not.toContain("PRIVATE");
  expect(JSON.stringify(telemetry.sentry)).not.toContain("PRIVATE");
});

for (const preference of ["stored", "gpc", "dnt"]) {
  test(`${preference} disables both browser providers before initialization`, async ({
    page,
    telemetry,
  }) => {
    await page.addInitScript((preference) => {
      if (preference === "stored")
        localStorage.setItem("burrow.telemetry.optout", "true");
      if (preference === "gpc")
        Object.defineProperty(navigator, "globalPrivacyControl", {
          value: true,
        });
      if (preference === "dnt")
        Object.defineProperty(navigator, "doNotTrack", { value: "1" });
    }, preference);
    await page.goto("/");
    await expect(
      page.getByRole("button", {
        name:
          preference === "stored"
            ? "Enable website telemetry"
            : "Telemetry disabled by browser",
      }),
    ).toBeVisible();
    await page
      .getByRole("link", { name: "Privacy policy", exact: true })
      .click();
    await expect(page.getByRole("heading", { level: 1 })).toContainText(
      "Privacy",
    );
    expect(telemetry.requests).toEqual([]);
  });
}

test("opt-out clears identity, persists and blocks already initialized SDK transports", async ({
  page,
  context,
  telemetry,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/");
  await expect
    .poll(() => telemetry.events.some((event) => event.event === "$pageview"), {
      timeout: 15000,
    })
    .toBe(true);
  const identity = telemetry.events.find((event) => event.event === "$pageview")
    ?.properties.distinct_id;
  await page.getByRole("button", { name: "Disable website telemetry" }).click();
  expect(
    await page.evaluate(() =>
      Object.keys(localStorage).filter(
        (key) => key.startsWith("ph_") && key.endsWith("_posthog"),
      ),
    ),
  ).toEqual([]);
  telemetry.requests.length = 0;
  await page
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  await page.evaluate(() => {
    setTimeout(() => {
      throw new Error("opted out");
    }, 0);
  });
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Enable website telemetry" }),
  ).toBeVisible();
  expect(telemetry.requests).toEqual([]);
  await page.getByRole("button", { name: "Enable website telemetry" }).click();
  await expect
    .poll(
      () =>
        telemetry.events.filter((event) => event.event === "$pageview").length,
      { timeout: 15000 },
    )
    .toBe(2);
  expect(
    telemetry.events.filter((event) => event.event === "$pageview")[1]
      .properties.distinct_id,
  ).not.toBe(identity);
});

test("clipboard failure is an outcome rather than an installation conversion", async ({
  page,
  telemetry,
}) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: () => Promise.reject(new Error("denied")) },
    });
  });
  await page.goto("/");
  await page
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  await expect(
    page.getByText(
      "Clipboard unavailable. Select and copy the command instead.",
    ),
  ).toBeVisible();
  await expect
    .poll(
      () =>
        telemetry.events.some((event) => event.event === "operation_completed"),
      { timeout: 15000 },
    )
    .toBe(true);
  expect(
    telemetry.events.some((event) => event.event === "install_command_copied"),
  ).toBe(false);
});

test("cross-tab re-enable never revives a cleared identity", async ({
  page,
  context,
  telemetry,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/");
  const other = await context.newPage();
  await other.goto("/");
  const views = () =>
    telemetry.events.filter((event) => event.event === "$pageview");
  await expect.poll(() => views().length, { timeout: 15000 }).toBe(2);
  const oldIdentity = views()[0].properties.distinct_id;
  await page.getByRole("button", { name: "Disable website telemetry" }).click();
  await expect(
    other.getByRole("button", { name: "Enable website telemetry" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Enable website telemetry" }).click();
  await expect.poll(() => views().length, { timeout: 15000 }).toBe(3);
  const before = telemetry.events.length;
  await other
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  await other.evaluate(() => {
    setTimeout(() => {
      throw new Error("cleared identity");
    }, 0);
  });
  await page
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  await expect
    .poll(
      () =>
        telemetry.events
          .slice(before)
          .filter((event) => event.event === "install_command_copied").length,
      { timeout: 15000 },
    )
    .toBe(1);
  expect(
    telemetry.events
      .slice(before)
      .some((event) => event.properties.distinct_id === oldIdentity),
  ).toBe(false);
});

test("a failed preference write disables collection in the current tab", async ({
  page,
  context,
  telemetry,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/");
  await expect
    .poll(() => telemetry.events.some((event) => event.event === "$pageview"), {
      timeout: 15000,
    })
    .toBe(true);
  await page.evaluate(() => {
    Storage.prototype.setItem = () => {
      throw new DOMException("Storage full", "QuotaExceededError");
    };
  });
  await page.getByRole("button", { name: "Disable website telemetry" }).click();
  await expect(
    page.getByRole("button", { name: "Enable website telemetry" }),
  ).toBeVisible();
  telemetry.requests.length = 0;
  await page
    .getByRole("button", { name: "Copy npm install command", exact: true })
    .click();
  // Keep the injected write failure active: a full navigation creates a fresh
  // document with healthy storage. Exercise the in-page action instead.
  await page.locator("[data-analytics-event=cta_clicked]").click();
  await expect(page).toHaveURL(/#install$/);
  expect(telemetry.requests).toEqual([]);
});

test("provider failures leave the product usable", async ({
  page,
  context,
}) => {
  await context.route(/^https:\/\/[^/]*(posthog\.com|sentry\.io)\//, (route) =>
    route.abort(),
  );
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/");
  await page.locator("[data-analytics-event=cta_clicked]").click();
  await expect(page).toHaveURL(/#install$/);
  await page
    .getByRole("button", { name: "Copy install the cli command", exact: true })
    .click();
  await expect(
    page.getByText("Copied to clipboard.", { exact: true }),
  ).toBeVisible();
  expect(errors).toEqual([]);
});

test.describe("remote controls", () => {
  test.use({ telemetrySettings: { replay: true, variant: "benefit" } });
  test("decoded recordings mask text, inputs, media and inherited query strings", async ({
    page,
    telemetry,
  }) => {
    await page.goto("/?token=REPLAY_SECRET#REPLAY_SECRET");
    await expect(
      page.getByRole("link", { name: "Share your local app", exact: true }),
    ).toBeVisible();
    await expect
      .poll(() =>
        telemetry.requests.some((url) => url.includes("posthog-recorder.js")),
      )
      .toBe(true);
    await page.evaluate(() => {
      const content = document.createElement("section");
      content.innerHTML =
        '<p>REPLAY_SECRET</p><input id="replay-input" value="REPLAY_SECRET"/><div data-private>REPLAY_SECRET</div><img src="/REPLAY_SECRET.png" alt="REPLAY_SECRET" />';
      document.getElementById("main")!.append(content);
    });
    await page
      .getByRole("link", { name: "Share your local app", exact: true })
      .click();
    await expect
      .poll(
        () =>
          JSON.stringify(
            telemetry.events.filter((event) => event.event === "$snapshot"),
          ).includes("replay-input"),
        { timeout: 20000 },
      )
      .toBe(true);
    expect(
      JSON.stringify(
        telemetry.events.filter((event) => event.event === "$snapshot"),
      ),
    ).not.toContain("REPLAY_SECRET");
    const exposures = telemetry.events.filter(
      (event) =>
        event.event === "$feature_flag_called" &&
        event.properties.$feature_flag === "website-cta-copy",
    );
    expect(exposures).toHaveLength(1);
    expect(exposures[0].properties.$feature_flag_response).toBe("benefit");
  });
});
