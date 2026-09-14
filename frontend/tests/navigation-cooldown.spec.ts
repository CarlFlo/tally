import { expect, test } from "@playwright/test";

test("route changes apply a 100 ms cooldown after four navigations", async ({ page }) => {
  await page.clock.install();
  await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" },
    data: { profile: "user0" },
  });
  await page.goto("/settings");
  await page.getByRole("link", { name: "Torrent client", exact: true }).click();
  await expect(page).toHaveURL("/settings/torrent");
  await expect(page.locator(".app-shell")).not.toHaveClass(/navigation-cooling/);

  await page.getByRole("link", { name: "Torrent search", exact: true }).last().click();
  await expect(page).toHaveURL("/settings/search");
  await expect(page.locator(".app-shell")).not.toHaveClass(/navigation-cooling/);

  await page.getByRole("link", { name: "Notifications", exact: true }).last().click();
  await expect(page).toHaveURL("/settings/notifications");
  await expect(page.locator(".app-shell")).not.toHaveClass(/navigation-cooling/);

  await page.getByRole("link", { name: "Bell Notifications", exact: true }).last().click();
  await expect(page).toHaveURL("/settings/bell");
  await expect(page.locator(".app-shell")).toHaveClass(/navigation-cooling/);

  const searchLink = page.getByRole("link", { name: "Torrent search", exact: true }).last();
  await expect(searchLink).toHaveCSS("pointer-events", "none");
  await expect(page).toHaveURL("/settings/bell");

  await page.clock.fastForward(100);
  await expect(page.locator(".app-shell")).not.toHaveClass(/navigation-cooling/);
  await page.clock.fastForward(300);
  await page.getByRole("link", { name: "Torrent search", exact: true }).last().click();
  await expect(page).toHaveURL("/settings/search");
  await expect(page.locator(".app-shell")).not.toHaveClass(/navigation-cooling/);
});
