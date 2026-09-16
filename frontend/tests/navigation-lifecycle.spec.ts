import { expect, test } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("200 route changes remain interactive with bounded resources and no cooldown", async ({ page }) => {
  test.setTimeout(180_000);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.addInitScript(() => {
    const state = { streams: 0, peakStreams: 0, requests: 0, peakRequests: 0, intervals: new Set<number>() };
    (window as any).lifecycle = state;
    const Base = window.EventSource;
    window.EventSource = class extends Base {
      private closed = false;
      constructor(url: string | URL, options?: EventSourceInit) {
        super(url, options);
        state.peakStreams = Math.max(state.peakStreams, ++state.streams);
      }
      close() {
        if (!this.closed) { state.streams--; this.closed = true; }
        super.close();
      }
    };
    const fetch = window.fetch.bind(window);
    window.fetch = async (...args) => {
      state.peakRequests = Math.max(state.peakRequests, ++state.requests);
      try { return await fetch(...args); } finally { state.requests--; }
    };
    const interval = window.setInterval.bind(window), clear = window.clearInterval.bind(window);
    window.setInterval = ((...args: Parameters<typeof interval>) => {
      const id = interval(...args); state.intervals.add(id); return id;
    }) as typeof interval;
    window.clearInterval = (id) => { state.intervals.delete(id!); clear(id); };
  });
  await selectProfileByName(page, "My profile");
  await page.goto("/settings");
  await expect(page.locator(".schedule-editor")).toHaveCount(3);
  // Bootstrap/localization loading is intentionally outside this test's scope.
  // Measure only the request concurrency caused by client-side navigation.
  await expect.poll(() => page.evaluate(() => (window as any).lifecycle.requests)).toBe(0);
  await page.evaluate(() => {
    (window as any).lifecycle.peakRequests = 0;
  });
  const original = await page.evaluateHandle(() => document);
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Performance.enable");
  async function resources() {
    await cdp.send("HeapProfiler.collectGarbage");
    const { metrics } = await cdp.send("Performance.getMetrics");
    return metrics.find((entry: any) => entry.name === "JSEventListeners")!.value;
  }
  let baseline = 0;
  for (let cycle = 0; cycle < 21; cycle++) {
    // Router links and browser history share the same document. No reloads.
    for (const path of ["/settings/torrent", "/settings/search", "/settings/notifications", "/settings/bell", "/settings"]) {
      await page.locator(`.settings-tabs a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }
    await page.getByRole("button", { name: "Open profile menu" }).click();
    await page.locator('#profile-menu a[href="/system"]').click();
    for (const path of ["/system/statistics", "/system/logs", "/system/jobs"]) {
      await page.locator(`.system-tabs a[href="${path}"]`).click();
    }
    await page.getByRole("button", { name: "Open profile menu" }).click();
    await page.locator('#profile-menu a[href="/settings"]').click();
    await expect(page.locator(".schedule-editor")).toHaveCount(3);
    if (cycle === 0) baseline = await resources();
  }
  expect(await resources()).toBeLessThanOrEqual(baseline + 5);
  expect(await original.evaluate((doc) => doc === document)).toBe(true);
  await page.goBack();
  await page.goForward();
  await page.locator('.settings-tabs a[href="/settings/search"]').click();
  await page.getByLabel("Jackett base URL", { exact: true }).fill("http://still-interactive.invalid");
  await expect(page.getByLabel("Jackett base URL", { exact: true })).toHaveValue("http://still-interactive.invalid");
  await expect(page.locator(".navigation-cooling, [inert], dialog[open]")).toHaveCount(0);
  await expect.poll(() => page.evaluate(() => (window as any).lifecycle.requests)).toBe(0);
  const state = await page.evaluate(() => {
    const s = (window as any).lifecycle;
    return { streams: s.streams, peakStreams: s.peakStreams, peakRequests: s.peakRequests, intervals: s.intervals.size };
  });
  expect(state).toEqual({ streams: 1, peakStreams: 1, peakRequests: expect.any(Number), intervals: 0 });
  expect(state.peakRequests).toBeLessThanOrEqual(6);
  expect(errors).toEqual([]);
});
