test("localization loading failure can be retried", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  let failures = 0;
  await page.route("**/api/locales", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/locales" && failures < 2) {
      failures++;
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "temporary locale failure" }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto("/calendar");
  await expect(
    page.getByText("Localization could not be loaded.", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
});

import { expect, test } from "@playwright/test";
import { openProfile, selectProfileByName } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

test.afterEach(async ({ page }) => {
  await selectProfileByName(page, "My profile");
  const response = await page.request.get("/api/bootstrap");
  expect(response.ok()).toBe(true);
  const boot = await response.json();
  if (boot.profile?.locale === "en") return;
  const restore = await page.request.patch("/api/profile", {
    headers,
    data: {
      name: boot.profile.display_name,
      avatar: boot.profile.avatar,
      locale: "en",
    },
  });
  expect(restore.ok()).toBe(true);
});

test("profile language applies on save and persists per profile", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();

  await openProfile(page);
  const language = page.getByLabel("Language", { exact: true });
  await language.selectOption("zz-Test");

  // Selecting a language only changes the draft profile value. The active
  // locale remains the saved profile locale until Save profile is clicked.
  await expect(
    page.getByRole("heading", { name: "My profile." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.locator("html")).toHaveAttribute("dir", "ltr");

  // Leaving without saving discards the draft language and keeps English active.
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");

  await openProfile(page);
  await expect(language).toHaveValue("en");
  await language.selectOption("zz-Test");
  await page.getByRole("button", { name: "Save profile", exact: true }).click();

  // The saved profile locale becomes authoritative after bootstrap refreshes.
  await expect(page.getByRole("status")).toContainText("Test profile updated");
  await expect(
    page.getByRole("heading", { name: "Test profile." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "zz-Test");
  await expect(page.locator("html")).toHaveAttribute("dir", "ltr");

  // Untranslated keys in this deliberately partial fixture still fall back to English.
  // The staged save bar disappears once the profile is clean.
  await expect(
    page.getByRole("button", { name: "Save profile", exact: true }),
  ).toHaveCount(0);

  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Test calendar." }),
  ).toBeVisible();

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "zz-Test");
  await expect(
    page.getByRole("heading", { name: "Test calendar." }),
  ).toBeVisible();

  // A new profile still defaults to English, even while the administrator's
  // saved profile locale is the test locale.
  await page.goto("/admin/access/profiles");
  await page.getByRole("button", { name: "New profile", exact: true }).click();
  const createDialog = page.getByRole("dialog", {
    name: "A new personal space",
    exact: true,
  });
  await expect(createDialog).toBeVisible();
  await expect(createDialog.getByLabel("Language", { exact: true })).toHaveValue(
    "en",
  );
  await createDialog.getByRole("button", { name: "Close dialog" }).click();

  // Restore the shared browser fixture for the rest of the serial suite.
  await openProfile(page);
  await page.getByLabel("Language", { exact: true }).selectOption("en");
  await expect(
    page.getByRole("heading", { name: "Test profile." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "zz-Test");
  await page.getByRole("button", { name: "Save profile", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "My profile." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
});
