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

export async function profileId(page: Page, name: string) {
  const boot = await (await page.request.get("/api/bootstrap")).json();
  const profile = boot.profiles.find((item: any) => item.display_name === name);
  if (!profile) throw new Error(`Profile not found: ${name}`);
  return profile.id as string;
}

export async function selectProfileByName(page: Page, name: string) {
  const id = await profileId(page, name);
  const response = await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" },
    data: { profile: id },
  });
  expect(response.ok()).toBe(true);
  return id;
}
