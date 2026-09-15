import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("discovery focus, inside clicks, queued add/undo, retry notice, favorites and episode toggles", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  const headers = { "X-Tally-CSRF": "1" };
  await selectProfileByName(page, "My profile");
  const candidates = [70, 71, 72, 73].map((id) => ({
    show: {
      id,
      name: `Fixture show ${id}`,
      premiered: "2026-01-01",
      status: "Running",
      summary: "A recent, highly rated story for the local browser fixture.",
      genres: ["Drama"],
      rating: { average: 8.5 },
    },
    followed: false,
  }));
  await page.route("**/api/shows/suggestions", (r) =>
    r.fulfill({ json: candidates }),
  );
  let release!: () => void;
  const gate = new Promise<void>((r) => (release = r));
  let rejected = false;
  await page.route("**/api/show-actions", async (route) => {
    if (route.request().method() !== "POST") return route.continue();
    const data = route.request().postDataJSON();
    if (data.tvmaze_id === 70 && data.follow) await gate;
    if (data.tvmaze_id === 72 && data.follow && !rejected) {
      rejected = true;
      return route.fulfill({
        status: 503,
        json: { error: "Fixture add failed; please retry" },
      });
    }
    await route.continue();
  });
  await page.goto("/calendar");
  await page.getByRole("button", { name: "Add show", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Find your next favorite" });
  const search = dialog.getByRole("textbox", { name: "Search for a TV show" });
  await expect(search).toBeFocused();
  await expect(dialog.locator(".search-show-card")).toHaveCount(4);
  const box = await dialog.boundingBox();
  await page.mouse.click(box!.x + 4, box!.y + box!.height / 2);
  await expect(dialog).toBeVisible();
  await dialog
    .getByRole("button", { name: "About Fixture show 70", exact: true })
    .click();
  const details = page.getByRole("dialog", {
    name: "Fixture show 70",
    exact: true,
  });
  await details.getByRole("button", { name: "Add", exact: true }).click();
  await expect(
    details.getByRole("button", { name: "Added", exact: true }),
  ).toBeEnabled();
  await details.getByRole("button", { name: "Close dialog" }).click();
  const card = (id: number) =>
    dialog.locator(".search-show-card").filter({
      has: page.getByRole("heading", { name: `Fixture show ${id} 2026` }),
    });
  await card(71).hover();
  await card(71).getByRole("button", { name: "Add", exact: true }).click();
  await expect(
    card(71).getByRole("button", { name: "Added", exact: true }),
  ).toBeEnabled();
  await card(70).hover();
  await card(70).getByRole("button", { name: "Added", exact: true }).click();
  await expect(
    card(70).getByRole("button", { name: "Add", exact: true }),
  ).toBeVisible();
  release();
  await card(72).hover();
  await card(72).getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("Fixture add failed");
  const toast = await page.getByRole("alert").boundingBox();
  expect(toast!.y).toBeLessThan(80);
  await expect(
    card(72).getByRole("button", { name: "Add", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(
    card(72).getByRole("button", { name: "Added", exact: true }),
  ).toBeVisible();
  await page.screenshot({
    path: "../docs/screenshots/discovery-desktop.png",
    fullPage: true,
  });
  await search.fill("Example");
  await expect(
    dialog.getByRole("heading", { name: "Fixture show 73 2026" }),
  ).toHaveCount(0);
  await expect(
    dialog.getByRole("heading", { name: "Example Show 2026" }),
  ).toBeVisible();
  await search.fill("");
  await expect(card(73)).toBeVisible();
  const bounds = await dialog.boundingBox();
  await page.mouse.click(Math.max(1, bounds!.x - 8), bounds!.y + 30);
  await expect(dialog).toHaveCount(0);
  await expect
    .poll(async () => {
      const actions = await (
        await page.request.get("/api/show-actions")
      ).json();
      return actions
        .filter((a: any) => [70, 71, 72].includes(a.external_id))
        .every((a: any) => a.status === "done");
    })
    .toBe(true);
  await page.goto("/shows");
  await expect(
    page.getByRole("heading", { name: "Fixture show 70", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Favorite Fixture show 71", exact: true })
    .click();
  await expect(page.getByRole("region", { name: "Favorites" })).toContainText(
    "Fixture show 71",
  );
  await page.reload();
  await expect(
    page.getByRole("button", {
      name: "Unfavorite Fixture show 71",
      exact: true,
    }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .getByRole("region", { name: "Favorites" })
    .getByRole("link")
    .click();
  const episode = page.locator(".episode-row").first();
  await episode
    .getByRole("button", { name: "Mark downloaded", exact: true })
    .click();
  await expect(
    episode.getByRole("button", { name: "Mark not downloaded", exact: true }),
  ).toBeEnabled();
  await episode
    .getByRole("button", { name: "Mark not downloaded", exact: true })
    .click();
  await expect(
    episode.getByRole("button", { name: "Mark downloaded", exact: true }),
  ).toBeEnabled();
  await episode
    .getByRole("button", { name: "Mark watched", exact: true })
    .click();
  await expect(
    episode.getByRole("button", { name: "Mark unwatched", exact: true }),
  ).toBeEnabled();
  await episode
    .getByRole("button", { name: "Mark unwatched", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.reload();
  await expect(
    episode.getByRole("button", { name: "Mark downloaded", exact: true }),
  ).toHaveAttribute("aria-pressed", "false");
  await page
    .getByRole("button", { name: "Mark season downloaded", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Clear downloaded season", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Mark season downloaded", exact: true }),
  ).toBeEnabled();
  await page.goto("/calendar");
  await page.getByRole("button", { name: "Month", exact: true }).click();
  await expect(page.locator(".calendar-favorite").first()).toBeVisible();
  await expect(page.getByText("Your rotation", { exact: true })).toHaveCount(0);
  await expect(
    page.getByText("Your pace. Your space.", { exact: true }),
  ).toHaveCount(0);
  const arrow = page.getByRole("button", { name: "Next period" });
  const initial = await arrow.boundingBox();
  for (let i = 0; i < 4; i++) {
    await arrow.click();
    const after = await arrow.boundingBox();
    expect(after!.x).toBeCloseTo(initial!.x, 0);
  }
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "Add show", exact: true }).click();
  await expect(search).toBeFocused();
  await expect(dialog.locator(".search-show-card")).toHaveCount(4);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/discovery-mobile.png",
    fullPage: true,
  });
  expect(errors).toEqual([]);
});

test("settings categories persist connections, schedules, debug previews and statistics limits", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  const headers = { "X-Tally-CSRF": "1" };
  await selectProfileByName(page, "My profile");
  const deploymentSettings = await (await page.request.get("/api/settings")).json();
  const notificationSettings = await (
    await page.request.get("/api/settings/notifications")
  ).json();
  expect(notificationSettings.data.timezone).toBe(deploymentSettings.timezone);
  const initialBoot = await (await page.request.get("/api/bootstrap")).json();
  const originalTimezone = initialBoot.preferences.timezone;
  const userTimezone =
    deploymentSettings.timezone === "America/New_York"
      ? "Europe/Stockholm"
      : "America/New_York";
  await page.request.patch("/api/preferences", {
    headers,
    data: { timezone: userTimezone, time_format: "24h" },
  });
  await page.goto("/settings/notifications");
  await expect(page.getByLabel("Notification timezone")).toHaveCount(0);
  await expect(page.locator(".notification-timezone-chip")).toHaveText(
    deploymentSettings.timezone,
  );
  const deliveryInput = page.getByLabel("Daily release notification time");
  await deliveryInput.fill("09:00");
  const deliveryPreview = await (
    await page.request.post("/api/settings/notifications/preview", {
      headers,
      data: { delivery_time: "09:00" },
    })
  ).json();
  const expectedUserTime = new Intl.DateTimeFormat("en-GB", {
    timeZone: userTimezone,
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(deliveryPreview.next_delivery * 1000));
  await expect(page.locator(".notification-local-time")).toContainText(
    `Your time: ${expectedUserTime} · ${userTimezone}`,
  );
  await page
    .getByLabel("Webhook URL", { exact: true })
    .fill("http://127.0.0.1:1/fixture-webhook");
  await page
    .getByRole("button", { name: "Save settings", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Notification settings saved",
  );
  await page.getByRole("switch", { name: "Enable all notifications" }).check();
  await expect(page.getByRole("status")).toContainText("Notifications enabled");
  await page.reload();
  await expect(
    page.getByRole("switch", { name: "Enable all notifications" }),
  ).toBeChecked();
  await expect(page.getByLabel("Webhook URL", { exact: true })).toHaveValue(
    "http://127.0.0.1:1/fixture-webhook",
  );
  await page
    .getByRole("switch", { name: "Enable all notifications" })
    .uncheck();
  await expect(page.getByRole("status")).toContainText(
    "All notifications disabled",
  );
  await page
    .getByRole("link", { name: "Torrent search", exact: true })
    .last()
    .click();
  await expect(page).toHaveURL(/\/settings\/search$/);
  await page
    .getByLabel("Jackett base URL", { exact: true })
    .fill("http://127.0.0.1:1");
  await page
    .getByLabel("API key", { exact: true })
    .fill("plain-fixture-key");
  await page
    .getByRole("checkbox", { name: "Enable Jackett search" })
    .uncheck();
  await page
    .getByRole("button", { name: "Save Jackett", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Jackett disabled");
  await page.reload();
  const key = page.getByLabel("API key", { exact: true });
  await expect(key).toHaveValue("plain-fixture-key");
  await expect(key).toHaveAttribute("type", "text");
  await expect(key).toHaveAttribute("autocomplete", "off");
  await page
    .getByRole("link", { name: "Scheduling & backups", exact: true })
    .click();
  const schedule = page.locator(".schedule-editor").filter({
    has: page.getByRole("heading", { name: "Maintenance", exact: true }),
  });
  await schedule
    .getByLabel("Cron expression")
    .fill("15 8 * * 1-5");
  await schedule.getByRole("checkbox", { name: "Run automatically" }).uncheck();
  await schedule
    .getByRole("button", { name: "Save schedule", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Schedule saved");
  await page.reload();
  await expect(
    schedule.getByLabel("Cron expression"),
  ).toHaveValue("15 8 * * 1-5");
  await expect(
    schedule.getByRole("checkbox", { name: "Run automatically" }),
  ).not.toBeChecked();
  await page.getByRole("link", { name: "Debug", exact: true }).click();
  const environment = page.locator(".environment-settings");
  await expect(environment).toContainText("TZ");
  await expect(environment).toContainText(deploymentSettings.timezone);
  await expect(page.getByText(/Secrets and credential values are intentionally not shown/)).toBeVisible();
  await page.getByRole("checkbox", { name: "Enable debug mode" }).check();
  await expect
    .poll(async () => {
      const boot = await (await page.request.get("/api/bootstrap")).json();
      return boot.preferences.debug_mode;
    })
    .toBe(true);
  await page.goto("/system/jobs");
  await expect(
    page.getByRole("button", { name: "Preview paused", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Preview paused", exact: true })
    .click();
  await expect(
    page.getByText("Paused after 3 consecutive failures."),
  ).toBeVisible();
  await expect(
    page.locator(".job-card").filter({
      has: page.getByRole("heading", { name: "Maintenance", exact: true }),
    }),
  ).toHaveClass(/is-disabled/);
  await page
    .getByRole("combobox", { name: "Filter history by job" })
    .selectOption("backup");
  await page
    .getByRole("combobox", { name: "Filter history by result" })
    .selectOption("failed");
  await expect
    .poll(async () => {
      const boot = await (await page.request.get("/api/bootstrap")).json();
      return {
        job: boot.preferences.job_type_filter,
        status: boot.preferences.job_status_filter,
      };
    })
    .toEqual({ job: "backup", status: "failed" });
  await page.reload();
  await expect(
    page.getByRole("combobox", { name: "Filter history by job" }),
  ).toHaveValue("backup");
  await expect(
    page.getByRole("combobox", { name: "Filter history by result" }),
  ).toHaveValue("failed");
  await page.request.patch("/api/preferences", {
    headers,
    data: { time_format: "12h" },
  });
  await page.reload();
  await expect(
    page.getByText("At 08:15, Monday through Friday"),
  ).toBeVisible();
  await page.screenshot({
    path: "../docs/screenshots/jobs-previews-desktop.png",
    fullPage: true,
  });
  await page.goto("/statistics");
  await expect(
    page.getByRole("combobox", { name: "Recent requests rows" }),
  ).toHaveValue("20");
  await page
    .getByRole("combobox", { name: "Recent requests rows" })
    .selectOption("50");
  await page
    .getByRole("combobox", { name: "Next metadata checks rows" })
    .selectOption("100");
  await expect
    .poll(async () => {
      const boot = await (await page.request.get("/api/bootstrap")).json();
      return [boot.preferences.request_limit, boot.preferences.scan_limit];
    })
    .toEqual([50, 100]);
  await page.reload();
  await expect(
    page.getByRole("combobox", { name: "Recent requests rows" }),
  ).toHaveValue("50");
  await expect(
    page.getByRole("combobox", { name: "Next metadata checks rows" }),
  ).toHaveValue("100");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/settings/scheduling");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/scheduling-mobile.png",
    fullPage: true,
  });
  await page.request.patch("/api/preferences", {
    headers,
    data: {
      time_format: "24h",
      debug_mode: false,
      debug_job_state: "normal",
      job_type_filter: "all",
      job_status_filter: "all",
      timezone: originalTimezone,
    },
  });
  expect(errors).toEqual([]);
});
