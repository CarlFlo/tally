import { test, expect, type Page } from "@playwright/test";
import { openProfileMenu, selectProfileByName } from "./navigation";

async function section(page: Page, name: string) {
  await openProfileMenu(page);
  await page.locator("#profile-menu").getByRole("link", { name, exact: true }).click();
}

test("continuous pointer navigation stays responsive without reloading", async ({ page }) => {
  test.setTimeout(180_000);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await selectProfileByName(page, "My profile");
  await page.setViewportSize({ width: 1100, height: 740 });
  await page.goto("/settings");
  await expect(page.locator(".schedule-editor")).toHaveCount(4);
  const documentHandle = await page.evaluateHandle(() => document);
  for (let cycle = 0; cycle < 3; cycle++) {
    // Exercise both sides of the sidebar breakpoint in the same document.
    await page.setViewportSize({ width: cycle % 2 ? 1024 : 1440, height: 740 });
    for (const path of ["/settings/torrent", "/settings/search", "/settings/notifications", "/settings/bell", "/settings/profiles", "/settings"]) {
      await page.locator(`.settings-tabs a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }
    await section(page, "System");
    for (const path of ["/system/statistics", "/system/logs", "/system/jobs"]) {
      await page.locator(`.system-tabs a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }
    for (const path of ["/calendar", "/shows", "/search"]) {
      await page.locator(`.sidebar nav a[href="${path}"]`).click();
      await expect(page).toHaveURL(path);
    }
    await page.getByRole("textbox", { name: "Torrent search query" }).fill(`cycle ${cycle}`);
    await section(page, "Settings");
  }
  expect(await documentHandle.evaluate((original) => original === document)).toBe(true);
  await page.locator('.settings-tabs a[href="/settings/search"]').click();
  await page.getByLabel("Jackett base URL", { exact: true }).fill("http://unchanged.invalid");
  await expect(page.getByLabel("Jackett base URL", { exact: true })).toHaveValue("http://unchanged.invalid");
  expect(errors).toEqual([]);
});
