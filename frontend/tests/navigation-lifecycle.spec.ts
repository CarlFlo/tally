import { expect, test } from "@playwright/test";
import { selectProfileByName } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

test("same-document navigation stays responsive with bounded resources during live updates", async ({
  page,
}) => {
  test.setTimeout(180_000);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });

  await page.addInitScript(() => {
    const state = {
      streams: 0,
      peakStreams: 0,
      requests: 0,
      peakRequests: 0,
      intervals: new Set<number>(),
    };
    (window as any).lifecycle = state;

    const Base = window.EventSource;
    window.EventSource = class extends Base {
      private closed = false;

      constructor(url: string | URL, options?: EventSourceInit) {
        super(url, options);
        state.peakStreams = Math.max(state.peakStreams, ++state.streams);
      }

      close() {
        if (!this.closed) {
          state.streams--;
          this.closed = true;
        }
        super.close();
      }
    };

    const fetch = window.fetch.bind(window);
    window.fetch = async (...args) => {
      state.peakRequests = Math.max(state.peakRequests, ++state.requests);
      try {
        return await fetch(...args);
      } finally {
        state.requests--;
      }
    };

    const interval = window.setInterval.bind(window);
    const clear = window.clearInterval.bind(window);
    window.setInterval = ((...args: Parameters<typeof interval>) => {
      const id = interval(...args);
      state.intervals.add(id);
      return id;
    }) as typeof interval;
    window.clearInterval = (id) => {
      state.intervals.delete(id!);
      clear(id);
    };
  });

  await selectProfileByName(page, "My profile");
  await page.setViewportSize({ width: 1100, height: 740 });
  await page.goto("/admin/configuration/schedules");
  await expect(page.locator(".schedule-editor")).toHaveCount(4);

  // Bootstrap/localization loading is intentionally outside this test's scope.
  await expect
    .poll(() => page.evaluate(() => (window as any).lifecycle.requests))
    .toBe(0);
  const startupPeakRequests = await page.evaluate(
    () => (window as any).lifecycle.peakRequests,
  );
  expect(startupPeakRequests).toBeLessThanOrEqual(6);
  await page.evaluate(() => {
    (window as any).lifecycle.peakRequests = 0;
  });

  const original = await page.evaluateHandle(() => document);
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Performance.enable");

  async function eventListeners() {
    await cdp.send("HeapProfiler.collectGarbage");
    const { metrics } = await cdp.send("Performance.getMetrics");
    return metrics.find((entry: any) => entry.name === "JSEventListeners")!
      .value;
  }

  const liveChanges = (async () => {
    for (let index = 0; index < 10; index++) {
      await page.request.patch("/api/preferences", {
        headers,
        data: { calendar_view: index % 2 ? "month" : "week" },
      });
      await page.waitForTimeout(170);
    }
  })();

  let baseline = 0;
  for (let cycle = 0; cycle < 21; cycle++) {
    // Exercise both sides of the sidebar breakpoint while keeping one document.
    await page.setViewportSize({
      width: cycle % 2 ? 1024 : 1440,
      height: 740,
    });

    for (const path of [
      "/admin/configuration/integrations/downloader",
      "/admin/configuration/integrations/search",
      "/admin/configuration/delivery",
      "/admin/advanced/diagnostics",
      "/admin/configuration/backups",
      "/admin/configuration/schedules",
    ]) {
      await page.locator(`.settings-tabs a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }

    await page.getByRole("button", { name: "Open profile menu" }).click();
    await page.locator('#profile-menu a[href="/admin/operations/jobs"]').click();
    for (const path of [
      "/admin/operations/statistics",
      "/admin/operations/logs",
      "/admin/operations/jobs",
    ]) {
      await page.locator(`.system-tabs a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }

    await page.getByRole("button", { name: "Open profile menu" }).click();
    await page.locator('#profile-menu a[href="/admin/configuration/schedules"]').click();
    await expect(page.locator(".schedule-editor")).toHaveCount(4);

    if (cycle === 0) baseline = await eventListeners();
  }

  await liveChanges;

  expect(await eventListeners()).toBeLessThanOrEqual(baseline + 5);
  expect(await original.evaluate((doc) => doc === document)).toBe(true);

  await page.goBack();
  await page.goForward();
  await page.locator('.settings-tabs a[href="/admin/configuration/integrations/search"]').click();
  await page
    .getByLabel("Jackett base URL", { exact: true })
    .fill("http://still-interactive.invalid");
  await expect(
    page.getByLabel("Jackett base URL", { exact: true }),
  ).toHaveValue("http://still-interactive.invalid");

  await expect(
    page.locator(".navigation-cooling, [inert], dialog[open]"),
  ).toHaveCount(0);
  await expect
    .poll(() => page.evaluate(() => (window as any).lifecycle.requests))
    .toBe(0);

  const state = await page.evaluate(() => {
    const current = (window as any).lifecycle;
    return {
      streams: current.streams,
      peakStreams: current.peakStreams,
      peakRequests: current.peakRequests,
      intervals: current.intervals.size,
    };
  });
  expect(state).toEqual({
    streams: 1,
    peakStreams: 1,
    peakRequests: expect.any(Number),
    intervals: 0,
  });
  expect(state.peakRequests).toBeLessThanOrEqual(6);
  expect(errors).toEqual([]);
});
