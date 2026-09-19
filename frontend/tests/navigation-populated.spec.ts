import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("populated System pages remain responsive across rapid navigation", async ({ page }) => {
  test.setTimeout(120_000);
  await selectProfileByName(page, "My profile");
  await page.addInitScript(() => {
    const original = Intl.DateTimeFormat;
    (window as any).formatterCount = 0;
    Intl.DateTimeFormat = new Proxy(original, {
      construct(target, args) {
        (window as any).formatterCount++;
        return Reflect.construct(target, args);
      },
    });
  });
  const now = Math.floor(Date.now() / 1000);
  await page.route("**/api/logs?*", (route) => route.fulfill({ json: {
    total: 50, actions: [], entries: Array.from({ length: 50 }, (_, id) => ({
      id, action: "job_succeeded", message: `Completed job ${id}`, created_at: now - id * 60, profile_name: "System",
    })),
  } }));
  await page.route("**/api/statistics?*", (route) => route.fulfill({ json: {
    summary: [], daily: [], states: [],
    requests: Array.from({ length: 100 }, (_, id) => ({ id, provider: "TVmaze", trigger: "scheduled_refresh", entity: `Show ${id}`, reason: "success", status_code: 200, duration_ms: 15, created_at: now - id * 60 })),
    next_scans: Array.from({ length: 100 }, (_, id) => ({ id, name: `Show ${id}`, next_check_at: now + id * 3600 })),
  } }));
  await page.goto("/system/logs");
  await expect(page.locator(".activity-entry")).toHaveCount(50);
  const started = Date.now();
  for (let i = 0; i < 40; i++) {
    await page.locator('.system-tabs a[href="/admin/operations/statistics"]').click();
    await page.locator('.system-tabs a[href="/admin/operations/logs"]').click();
  }
  console.log("Populated navigation", { elapsed: Date.now() - started, formatters: await page.evaluate(() => (window as any).formatterCount) });
  // Row count and route mounts must not allocate a formatter per date cell.
  expect(await page.evaluate(() => (window as any).formatterCount)).toBeLessThan(32);
  await page.getByRole("textbox", { name: "Search logs" }).fill("responsive");
  await expect(page.getByRole("textbox", { name: "Search logs" })).toHaveValue("responsive");
});
