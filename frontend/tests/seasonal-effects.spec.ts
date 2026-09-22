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
      ["2026-02-13T12:00:00Z", null],
      ["2026-02-14T12:00:00Z", "valentines-day"],
      ["2026-02-15T12:00:00Z", null],
      ["2026-03-31T12:00:00Z", null],
      ["2026-04-01T12:00:00Z", "april-fools-day"],
      ["2026-04-02T12:00:00Z", null],
      ["2026-04-02T12:00:00Z", null],
      ["2026-04-03T12:00:00Z", "easter"],
      ["2026-04-04T12:00:00Z", "easter"],
      ["2026-04-05T12:00:00Z", "easter"],
      ["2026-04-06T12:00:00Z", "easter"],
      ["2026-04-07T12:00:00Z", null],
      ["2026-06-18T12:00:00Z", null],
      ["2026-06-19T12:00:00Z", "swedish-midsummer"],
      ["2026-06-20T12:00:00Z", "swedish-midsummer"],
      ["2026-06-21T12:00:00Z", null],
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
      ["2026-12-12T12:00:00Z", null],
      ["2026-12-13T12:00:00Z", "st-lucia-day"],
      ["2026-12-14T12:00:00Z", null],
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

  test("updates a date-based effect after midnight without navigation", async ({
    page,
  }) => {
    await page.clock.install({
      time: new Date("2026-12-23T23:59:30Z"),
    });
    await selectProfileByName(page, "My profile");
    await page.goto("/calendar");

    const overlay = page.locator(".sidebar [data-seasonal-effect]");
    await expect(overlay).toHaveCount(0);

    await page.clock.fastForward(60_000);
    await expect(overlay).toHaveAttribute("data-seasonal-effect", "christmas");
  });

  test("keeps the logo layout stable when an overlay is enabled", async ({
    page,
  }) => {
    await page.clock.setFixedTime(new Date("2026-09-22T12:00:00Z"));
    await selectProfileByName(page, "My profile");
    await page.goto("/admin/advanced/diagnostics");

    const logo = page.locator(".sidebar .logo");
    const before = await logo.boundingBox();
    expect(before).not.toBeNull();

    await page.getByRole("combobox", { name: "Effect" }).selectOption("christmas");
    await page.getByRole("checkbox", { name: "Override seasonal effect" }).check();

    const after = await logo.boundingBox();
    expect(after).not.toBeNull();
    expect(after!.width).toBeCloseTo(before!.width, 1);
    expect(after!.height).toBeCloseTo(before!.height, 1);

    await page.evaluate(() => {
      document.documentElement.dataset.theme = "light";
    });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "Open navigation" }).click();
    await expect(
      page.locator('.sidebar [data-seasonal-effect="christmas"]'),
    ).toBeVisible();
  });

  test("keeps debug overrides usable when session storage is unavailable", async ({
    page,
  }) => {
    await page.addInitScript(() => {
      Object.defineProperty(window, "sessionStorage", {
        configurable: true,
        get() {
          throw new DOMException("Blocked for test", "SecurityError");
        },
      });
    });
    await page.clock.setFixedTime(new Date("2026-09-22T12:00:00Z"));
    await selectProfileByName(page, "My profile");
    await page.goto("/admin/advanced/diagnostics");

    await page
      .getByRole("combobox", { name: "Effect" })
      .selectOption("ukraine-independence-day");
    await page.getByRole("checkbox", { name: "Override seasonal effect" }).check();

    await expect(
      page.locator(
        '.sidebar [data-seasonal-effect="ukraine-independence-day"]',
      ),
    ).toBeVisible();
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
      "Valentine's Day",
      "April Fools' Day",
      "Easter",
      "Halloween",
      "Christmas",
      "New Year",
      "Sweden National Day",
      "Swedish Midsummer",
      "St. Lucia Day",
      "Ukraine Independence Day",
    ]);
    await expect(page.getByText("Applies: February 14")).toBeVisible();
    const overrideBox = await overrideToggle.boundingBox();
    const selectBox = await effectSelect.boundingBox();
    const datesBox = await page
      .getByText("Applies: February 14")
      .boundingBox();
    expect(overrideBox).not.toBeNull();
    expect(selectBox).not.toBeNull();
    expect(datesBox).not.toBeNull();
    expect(Math.abs(overrideBox!.y - selectBox!.y)).toBeLessThanOrEqual(6);
    expect(datesBox!.x).toBeGreaterThan(selectBox!.x + selectBox!.width);
    expect(Math.abs(datesBox!.y - selectBox!.y)).toBeLessThan(12);
    await expect(page.locator(".sidebar [data-seasonal-effect]")).toHaveCount(0);

    await effectSelect.selectOption("ukraine-independence-day");
    await expect(page.getByText("Applies: August 24")).toBeVisible();
    await expect(page.locator(".sidebar [data-seasonal-effect]")).toHaveCount(0);

    await effectSelect.selectOption("easter");
    await expect(page.getByText(/Applies: April 3.*6/)).toBeVisible();

    await effectSelect.selectOption("swedish-midsummer");
    await expect(page.getByText(/Applies: June 19.*20/)).toBeVisible();

    await effectSelect.selectOption("ukraine-independence-day");

    await overrideToggle.check();
    await expect(
      page.locator(
        '.sidebar [data-seasonal-effect="ukraine-independence-day"]',
      ),
    ).toBeVisible();

    await effectSelect.selectOption("christmas");
    await expect(page.getByText(/Applies: December 24.*26/)).toBeVisible();
    await expect(
      page.locator('.sidebar [data-seasonal-effect="christmas"]'),
    ).toBeVisible();

    await effectSelect.selectOption("new-year");
    await expect(
      page.getByText("Applies: December 31 and January 1"),
    ).toBeVisible();

    await effectSelect.selectOption("christmas");

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
