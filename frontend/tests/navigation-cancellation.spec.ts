import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("abandoned connection tests cannot disable or overwrite the next tab", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  let release!: () => void;
  const gate = new Promise<void>((resolve) => { release = resolve; });
  await page.route("**/api/settings/search/test", async (route) => {
    await gate;
    await route.fulfill({ json: { message: "Obsolete test result" } });
  });
  await page.goto("/admin/configuration/integrations/search");
  await page.getByLabel("Jackett base URL", { exact: true }).fill("http://fixture.invalid");
  await page.locator(".client-settings input.concealed-secret").fill("fixture-key");
  const pending = page.waitForRequest("**/api/settings/search/test");
  await page.getByRole("button", { name: "Test connection", exact: true }).click();
  const request = await pending;
  const aborted = page.waitForEvent("requestfailed", (failed) => failed === request);
  await page.locator('.settings-tabs a[href="/admin/configuration/backups"]').click();
  await aborted;
  release();
  await page.locator('.settings-tabs a[href="/admin/configuration/integrations/search"]').click();
  await page.getByLabel("Jackett base URL", { exact: true }).fill("http://new-draft.invalid");
  await expect(page.getByRole("button", { name: "Test connection", exact: true })).toBeEnabled();
  await expect(page.getByText("Obsolete test result", { exact: true })).toHaveCount(0);
  await expect(page.getByLabel("Jackett base URL", { exact: true })).toHaveValue("http://new-draft.invalid");
});

test("leaving a tab aborts its pending request and a later mount still loads", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  let release!: () => void;
  const gate = new Promise<void>((resolve) => { release = resolve; });
  let held = false;
  await page.route("**/api/statistics?*", async (route) => {
    if (held) return route.continue();
    held = true;
    await gate;
    await route.fulfill({ json: { summary: [], daily: [], states: [], requests: [], next_scans: [] } });
  });
  await page.goto("/admin/operations/jobs");
  await expect(page.locator(".job-card")).toHaveCount(4);
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
