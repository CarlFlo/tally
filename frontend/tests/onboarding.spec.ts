import { test, expect } from "@playwright/test";
import { signOut, openProfileMenu, profileId } from "./navigation";
test.use({ baseURL: "http://127.0.0.1:18082" });
const headers = { "X-Tally-CSRF": "1" };

test("profile creation and signed-out onboarding use the configured password", async ({
  page,
}) => {
  const adminID = await profileId(page, "My profile");
  await page.request.post("/api/auth/login", {
    headers,
    data: { profile: adminID, password: "1234" },
  });
  await page.goto("/admin/access/profiles");
  await page.getByRole("button", { name: "New profile", exact: true }).click();
  const evePassword = "Eve!1234";
  const createDialog = page.getByRole("dialog");
  await createDialog.getByLabel("Display name").fill("Eve");
  await createDialog.getByLabel("New password", { exact: true }).fill(evePassword);
  await createDialog.getByLabel("Confirm password", { exact: true }).fill(evePassword);
  await createDialog
    .getByRole("button", { name: "Create profile", exact: true })
    .click();
  await expect(page.locator(".profile-settings-list")).toContainText("Eve");
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("link", { name: "Add profile", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "EV Eve", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Welcome back, Eve." }),
  ).toBeVisible();
  await page.getByLabel("Password", { exact: true }).fill(evePassword);
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.locator(".header-profile strong")).toHaveText("Eve");
  let boot = await (await page.request.get("/api/bootstrap")).json();
  expect(
    (
      await page.request.delete(`/api/profiles/${boot.profile.id}`, { headers })
    ).status(),
  ).toBe(200);
  await page.goto("/login");
  await page.getByRole("link", { name: "Add profile", exact: true }).click();
  await expect(page).toHaveURL(/\/login\/new$/);
  await page.getByLabel("Display name").fill("Robin");
  await page.getByLabel("New password", { exact: true }).fill("Robin!1234");
  await page.getByLabel("Confirm password", { exact: true }).fill("Robin!1234");
  await page.screenshot({
    path: "../docs/screenshots/add-profile.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "Create profile", exact: true })
    .click();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.locator(".header-profile strong")).toHaveText("Robin");
  await openProfileMenu(page);
  await expect(
    page
      .locator("#profile-menu")
      .getByRole("link", { name: "System", exact: true }),
  ).toHaveCount(0);
  await expect(
    page
      .locator("#profile-menu")
      .getByRole("link", { name: "Settings", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.goto("/logs");
  await expect(page).toHaveURL(/\/logs$/);
  await expect(page.locator(".activity-list")).toContainText(
    "Robin created profile Robin",
  );
  await expect(page.locator(".activity-list")).not.toContainText("Eve");
  boot = await (await page.request.get("/api/bootstrap")).json();
  expect(
    (
      await page.request.delete(`/api/profiles/${boot.profile.id}`, { headers })
    ).status(),
  ).toBe(200);
});
