import { test, expect } from "@playwright/test";
import { signOut, openProfileMenu } from "./navigation";
test.use({ baseURL: "http://127.0.0.1:18082" });
const headers = { "X-Tally-CSRF": "1" };

test("passwordless profiles set their own password and signed-out visitors can add a profile", async ({
  page,
}) => {
  await page.request.post("/api/auth/login", {
    headers,
    data: { profile: "user0", password: "1234" },
  });
  await page.goto("/settings/profiles");
  await page.getByRole("button", { name: "New profile", exact: true }).click();
  await page.getByRole("dialog").getByLabel("Display name").fill("Eve");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Create profile", exact: true })
    .click();
  await expect(page.locator(".profile-settings-list")).toContainText("Eve");
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("link", { name: "Add profile", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "E Eve", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Set up your password, Eve." }),
  ).toBeVisible();
  const password = " MiXeD! 2468 ";
  await page.getByLabel("New password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password", { exact: true }).fill("different");
  await page.getByRole("button", { name: "Set password and continue" }).click();
  await expect(page.getByRole("alert")).toContainText("Passwords do not match");
  await page.getByLabel("Confirm password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Set password and continue" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.locator(".header-profile strong")).toHaveText("Eve");
  await signOut(page);
  await page.getByRole("button", { name: "E Eve", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Welcome back, Eve." }),
  ).toBeVisible();
  await page.getByLabel("Password or PIN", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
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
    "Created profile Robin",
  );
  await expect(page.locator(".activity-list")).not.toContainText("Eve");
  boot = await (await page.request.get("/api/bootstrap")).json();
  expect(
    (
      await page.request.delete(`/api/profiles/${boot.profile.id}`, { headers })
    ).status(),
  ).toBe(200);
});
