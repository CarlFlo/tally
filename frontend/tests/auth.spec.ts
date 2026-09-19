import { openProfile, signOut, profileId } from "./navigation";
import { test, expect } from "@playwright/test";

test.use({ baseURL: "http://127.0.0.1:18082" });

test("local sign-in follows browser history and switching requires sign-out", async ({
  page,
  browser,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const adminID = await profileId(page, "My profile");

  await page.goto("/calendar");
  await expect(page).toHaveURL(/\/login$/);
  await page.getByRole("button", { name: "MY My profile" }).click();
  await expect(page).toHaveURL(new RegExp("/login/" + adminID + "$"));
  await page.getByLabel("Password", { exact: true }).fill("unsent-password");
  await page.goBack();
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();
  await page.goForward();
  await expect(page).toHaveURL(new RegExp("/login/" + adminID + "$"));
  await expect(page.getByLabel("Password", { exact: true })).toHaveValue("");
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Welcome back, My profile." }),
  ).toBeVisible();
  await page.getByLabel("Password", { exact: true }).fill("1234");
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
  await openProfile(page);
  await expect(page).toHaveURL(/\/account$/);
  await page.getByRole("link", { name: "Security", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Your sessions" }),
  ).toBeVisible();
  await expect(page.getByText("This browser", { exact: true })).toBeVisible();

  const initialSessionCount = await page.locator(".session-row").count();
  const secondDevice = await browser.newContext();
  const secondDevicePage = await secondDevice.newPage();
  await secondDevicePage.goto("/login");
  await secondDevicePage.getByRole("button", { name: "MY My profile" }).click();
  await secondDevicePage.getByLabel("Password", { exact: true }).fill("1234");
  await secondDevicePage.getByRole("button", { name: "Enter your space" }).click();
  await expect(secondDevicePage).toHaveURL(/\/calendar$/);
  await expect(page.locator(".session-row")).toHaveCount(initialSessionCount + 1);
  await secondDevice.close();

  await page.goto("/admin/access/profiles");
  const currentProfile = page
    .locator(".profile-settings-list > div")
    .filter({ hasText: "My profile" });
  await expect(currentProfile.locator("small")).toContainText("Password protected");
  await page.getByRole("button", { name: "New profile", exact: true }).click();
  const createDialog = page.getByRole("dialog");
  await expect(createDialog.locator(".profile-create-preview .avatar.large")).toBeVisible();
  await expect(createDialog.locator(".avatar-choices button")).toHaveCount(6);
  await expect(
    createDialog.getByRole("combobox", { name: "Authentication", exact: true }),
  ).toBeVisible();
  await createDialog.getByRole("button", { name: "Cancel", exact: true }).click();

  const otherTab = await page.context().newPage();
  await otherTab.goto("/profile");
  await expect(
    otherTab.getByRole("textbox", { name: "Display name" }),
  ).toHaveValue("My profile");

  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await expect(otherTab).toHaveURL(/\/login$/);
  await expect(otherTab.locator(".app-shell")).toHaveCount(0);

  await page.goBack();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator(".app-shell")).toHaveCount(0);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();

  await page.getByRole("button", { name: "AL Alex" }).click();
  await page.getByLabel("Password", { exact: true }).fill("1234");
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page.locator(".header-profile strong")).toHaveText("Alex");
  await expect(otherTab).toHaveURL(/\/calendar$/);
  await expect(otherTab.locator(".header-profile strong")).toHaveText("Alex");

  await page.goto("/login/" + adminID);
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.locator(".header-profile strong")).toHaveText("Alex");
  expect(errors).toEqual([]);
  await otherTab.close();
});

test("new profile form has localized authentication copy and avatar preview", async ({
  page,
}) => {
  await page.goto("/login/new");
  await expect(page.locator(".profile-create-preview .avatar.large")).toBeVisible();
  await expect(page.locator(".avatar-choices button")).toHaveCount(6);
  const authentication = page.getByRole("combobox", {
    name: "Authentication",
    exact: true,
  });
  await expect(authentication).toBeVisible();
  await expect(authentication.locator("option")).toHaveText([
    "Password",
    "No authentication",
  ]);
  await expect(page.getByText("profile.authentication", { exact: true })).toHaveCount(0);
  await expect(page.getByText("profile.authPassword", { exact: true })).toHaveCount(0);
  await expect(page.getByText("profile.authNone", { exact: true })).toHaveCount(0);

  await authentication.selectOption("none");
  const warning = page.getByText(
    "Anyone who can reach Tally can enter this profile without a password.",
    { exact: true },
  );
  await expect(warning).toBeVisible();
  await expect(warning).toHaveClass(/auth-none-warning/);
});
