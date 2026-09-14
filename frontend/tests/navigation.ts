import { expect, type Page } from "@playwright/test";

export async function openProfileMenu(page: Page) {
  const button = page.getByRole("button", { name: "Open profile menu" });
  if ((await button.getAttribute("aria-expanded")) !== "true")
    await button.click();
  await expect(
    page.getByRole("navigation", { name: "Profile menu", exact: true }),
  ).toBeVisible();
}

export async function openProfile(page: Page) {
  await openProfileMenu(page);
  await page.getByRole("link", { name: "Profile", exact: true }).click();
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.locator("#profile-menu")).toHaveCount(0);
}

export async function signOut(page: Page) {
  await openProfileMenu(page);
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
}
