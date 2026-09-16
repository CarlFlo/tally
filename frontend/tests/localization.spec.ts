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

test("profile language previews immediately and persists per profile", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();

  await openProfile(page);
  const language = page.getByLabel("Language", { exact: true });
  await language.selectOption("sv");

  // Preview is immediate: no save, reload, or new session is needed.
  await expect(
    page.getByRole("heading", { name: "Min profil." }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "sv");
  await expect(page.locator("html")).toHaveAttribute("dir", "ltr");

  // Untranslated keys in this deliberately partial fixture still fall back to English.
  await expect(
    page.getByRole("button", { name: "Save profile", exact: true }),
  ).toBeVisible();

  await page.getByRole("button", { name: "Save profile", exact: true }).click();
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Din kalender." }),
  ).toBeVisible();

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "sv");
  await expect(
    page.getByRole("heading", { name: "Din kalender." }),
  ).toBeVisible();

  // A new profile defaults to English and previews that default immediately,
  // even while the administrator's saved profile locale is Swedish.
  await page.goto("/settings/profiles");
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
    page.getByRole("heading", { name: "My profile." }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save profile", exact: true }).click();
});
