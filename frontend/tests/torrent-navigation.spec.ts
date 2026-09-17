import { expect, test } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("pending torrent search does not block navigation to automation", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search");

  let requestStarted = false;
  await page.route("**/api/torrents/search", async (route) => {
    requestStarted = true;
    await new Promise((resolve) => setTimeout(resolve, 750));
    try {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ results: [], warnings: [] }),
      });
    } catch {
      // The route is expected to be cancelled when Search unmounts.
    }
  });

  const input = page.getByRole("textbox", { name: "Torrent search query" });
  await input.fill("Example Show S01E01");
  await page.getByRole("button", { name: "Search torrents", exact: true }).click();
  await expect.poll(() => requestStarted).toBe(true);

  await page.getByRole("link", { name: "Automation", exact: true }).click();
  await expect(page).toHaveURL(/\/search\/automation$/, { timeout: 1_500 });
  await expect(page.getByLabel("Minimum seeders", { exact: true })).toBeEnabled();

  await page.unroute("**/api/torrents/search");
});
