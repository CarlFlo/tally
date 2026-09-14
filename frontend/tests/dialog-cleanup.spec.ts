import { test, expect } from "@playwright/test";

test("navigation closes the actual modal element before discarding it", async ({ page }) => {
  await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" }, data: { profile: "user0" },
  });
  await page.goto("/calendar");
  await page.locator('.sidebar nav a[href="/shows"]').click();
  await page.getByRole("button", { name: "Add show", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  const dialog = await page.getByRole("dialog").elementHandle();
  await page.goBack();
  await expect(page).toHaveURL("/calendar");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(await dialog!.evaluate((element) => (element as HTMLDialogElement).open)).toBe(false);
  await page.getByRole("button", { name: "Add show", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "Search for a TV show" })).toBeFocused();
  await page.getByRole("button", { name: "Close dialog", exact: true }).click();
  await page.locator('.sidebar nav a[href="/search"]').click();
  await page.getByRole("textbox", { name: "Torrent search query" }).fill("still interactive");
});
