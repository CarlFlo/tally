import { test, expect } from "@playwright/test";

test("leaving a tab aborts its pending request and a later mount still loads", async ({ page }) => {
  await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" }, data: { profile: "user0" },
  });
  let release!: () => void;
  const gate = new Promise<void>((resolve) => { release = resolve; });
  let held = false;
  await page.route("**/api/statistics?*", async (route) => {
    if (held) return route.continue();
    held = true;
    await gate;
    await route.fulfill({ json: { summary: [], daily: [], states: [], requests: [], next_scans: [] } });
  });
  await page.goto("/system/jobs");
  await expect(page.locator(".job-card")).toHaveCount(3);
  const pending = page.waitForRequest("**/api/statistics?*");
  await page.getByRole("link", { name: "Statistics", exact: true }).click();
  const request = await pending;
  const aborted = page.waitForEvent("requestfailed", (failed) => failed === request);
  await page.getByRole("link", { name: "Logs", exact: true }).click();
  await aborted;
  release();
  await page.getByRole("textbox", { name: "Search logs" }).fill("unchanged filter");
  await expect(page.getByRole("textbox", { name: "Search logs" })).toHaveValue("unchanged filter");
  const loaded = page.waitForResponse("**/api/statistics?*");
  await page.getByRole("link", { name: "Statistics", exact: true }).click();
  expect((await loaded).ok()).toBe(true);
  await expect(page.getByRole("heading", { name: "Statistics" })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});
