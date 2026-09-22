import { expect, test } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test.describe("seasonal logo effects", () => {
  test.use({ timezoneId: "UTC" });

  test("uses local calendar dates for registered seasonal overlays", async ({
    page,
  }) => {
    await page.clock.setFixedTime(new Date("2026-09-22T12:00:00Z"));
    await selectProfileByName(page, "My profile");

    const cases: Array<[string, string | null]> = [
      ["2026-06-05T12:00:00Z", null],
      ["2026-06-06T12:00:00Z", "sweden-national-day"],
      ["2026-06-07T12:00:00Z", null],
      ["2026-08-23T12:00:00Z", null],
      ["2026-08-24T12:00:00Z", "ukraine-independence-day"],
      ["2026-08-25T12:00:00Z", null],
      ["2026-10-30T12:00:00Z", null],
      ["2026-10-31T12:00:00Z", "halloween"],
      ["2026-11-01T12:00:00Z", null],
      ["2026-12-23T12:00:00Z", null],
      ["2026-12-24T12:00:00Z", "christmas"],
      ["2026-12-25T12:00:00Z", "christmas"],
      ["2026-12-26T12:00:00Z", "christmas"],
      ["2026-12-27T12:00:00Z", null],
      ["2026-12-30T12:00:00Z", null],
      ["2026-12-31T12:00:00Z", "new-year"],
      ["2027-01-01T12:00:00Z", "new-year"],
      ["2027-01-02T12:00:00Z", null],
      ["2027-09-22T12:00:00Z", null],
    ];

    for (const [timestamp, expectedEffect] of cases) {
      await page.clock.setFixedTime(new Date(timestamp));
      await page.goto("/calendar");
      await expect(page.locator(".sidebar")).toBeVisible();

      const overlay = page.locator(".sidebar [data-seasonal-effect]");
      if (expectedEffect) {
        await expect(overlay).toHaveAttribute(
          "data-seasonal-effect",
          expectedEffect,
        );
      } else {
        await expect(overlay).toHaveCount(0);
      }
    }
  });

  test("debug override forces the selected effect for the browser session", async ({
    page,
  }) => {
    await page.clock.setFixedTime(new Date("2026-09-22T12:00:00Z"));
    await selectProfileByName(page, "My profile");
    await page.goto("/admin/advanced/diagnostics");

    const overrideToggle = page.getByRole("checkbox", {
      name: "Override seasonal effect",
    });
    const effectSelect = page.getByRole("combobox", { name: "Effect" });

    await expect(overrideToggle).not.toBeChecked();
    await expect(effectSelect.locator("option")).toHaveText([
      "Halloween",
      "Christmas",
      "New Year",
      "Sweden National Day",
      "Ukraine Independence Day",
    ]);
    await expect(page.locator(".sidebar [data-seasonal-effect]")).toHaveCount(0);

    await effectSelect.selectOption("ukraine-independence-day");
    await expect(page.locator(".sidebar [data-seasonal-effect]")).toHaveCount(0);

    await overrideToggle.check();
    await expect(
      page.locator(
        '.sidebar [data-seasonal-effect="ukraine-independence-day"]',
      ),
    ).toBeVisible();

    await effectSelect.selectOption("christmas");
    await expect(
      page.locator('.sidebar [data-seasonal-effect="christmas"]'),
    ).toBeVisible();

    await page.reload();
    await expect(overrideToggle).toBeChecked();
    await expect(effectSelect).toHaveValue("christmas");
    await expect(
      page.locator('.sidebar [data-seasonal-effect="christmas"]'),
    ).toBeVisible();

    await overrideToggle.uncheck();
    await expect(page.locator(".sidebar [data-seasonal-effect]")).toHaveCount(0);
  });
});
