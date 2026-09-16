import { expect, test } from "@playwright/test";
import { openProfile } from "./navigation";

test("profile language previews immediately and persists per profile", async ({
  page,
}) => {
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

  // Restore the shared browser fixture for the rest of the serial suite.
  await openProfile(page);
  await page.getByLabel("Language", { exact: true }).selectOption("en");
  await expect(
    page.getByRole("heading", { name: "My profile." }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save profile", exact: true }).click();
});
