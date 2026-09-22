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
    .getByRole("button", { name: "Unmark season downloaded", exact: true })
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

test("show season state toggles only released episodes", async ({ page }) => {
  await selectProfileByName(page, "My profile");

  const now = Date.now();
  const releasedStamp = new Date(now - 24 * 60 * 60 * 1000).toISOString();
  const futureStamp = new Date(now + 24 * 60 * 60 * 1000).toISOString();
  const state = {
    releasedWatched: false,
    releasedDownloaded: false,
  };
  const bulkBodies: any[] = [];
  const show = {
    favorite: 0,
    id: "future-state",
    name: "Release State Show",
    summary: "Fixture for release-state controls.",
    image: "",
    status: "Running",
    premiered: "2026-01-01",
    network: "Fixture",
    genres: "[]",
    rating: 0,
    runtime: 45,
    episode_count: 2,
    watched_count: 0,
    aired_count: 1,
    aired_unwatched: 1,
    next_episode: futureStamp.slice(0, 10),
    last_checked_at: 0,
    next_check_at: 0,
  };
  const episode = (
    id: string,
    number: number,
    stamp: string,
    watched: boolean,
    downloaded: boolean,
  ) => ({
    favorite: 0,
    id,
    show_id: show.id,
    show_name: show.name,
    show_image: "",
    name: id === "released-episode" ? "Released episode" : "Future episode",
    summary: "",
    season: 1,
    season_episode_count: 2,
    number,
    airdate: stamp.slice(0, 10),
    airstamp: stamp,
    runtime: 45,
    watched,
    downloaded,
    network: "Fixture",
    type: "regular",
  });

  await page.route("**/api/shows/future-state", async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        show: {
          ...show,
          watched_count: state.releasedWatched ? 1 : 0,
          aired_unwatched: state.releasedWatched ? 0 : 1,
        },
        episodes: [
          episode(
            "released-episode",
            1,
            releasedStamp,
            state.releasedWatched,
            state.releasedDownloaded,
          ),
          episode("future-episode", 2, futureStamp, false, false),
        ],
        seasons: [],
        external_ids: [],
      }),
    });
  });
  await page.route("**/api/shows/future-state/bulk", async (route) => {
    if (route.request().method() !== "POST") return route.continue();
    const body = route.request().postDataJSON();
    bulkBodies.push(body);
    if (typeof body.watched === "boolean") state.releasedWatched = body.watched;
    if (typeof body.downloaded === "boolean") state.releasedDownloaded = body.downloaded;
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ updated: 1 }),
    });
  });
  await page.route("**/api/torrents/automation/shows/future-state", async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ policy: "default", enabled: false }),
    });
  });

  await page.goto("/shows/future-state");
  await expect(page.getByRole("heading", { name: "Release State Show." })).toBeVisible();

  const releasedRow = page.locator(".episode-row").filter({ hasText: "Released episode" });
  const futureRow = page.locator(".episode-row").filter({ hasText: "Future episode" });
  await expect(
    releasedRow.getByRole("button", { name: "Mark watched", exact: true }),
  ).toBeEnabled();
  await expect(
    futureRow.getByRole("button", { name: "Mark watched", exact: true }),
  ).toBeDisabled();
  await expect(
    futureRow.getByRole("button", { name: "Mark downloaded", exact: true }),
  ).toBeDisabled();

  const watchedSeason = page.getByRole("button", {
    name: "Mark season watched",
    exact: true,
  });
  await watchedSeason.click();
  await expect(
    page.getByRole("button", { name: "Mark season unwatched", exact: true }),
  ).toBeVisible();
  expect(bulkBodies.at(-1)).toMatchObject({
    season: 1,
    aired_only: true,
    watched: true,
  });
  await expect(
    futureRow.getByRole("button", { name: "Mark watched", exact: true }),
  ).toBeDisabled();

  await page
    .getByRole("button", { name: "Mark season unwatched", exact: true })
    .click();
  await expect(watchedSeason).toBeVisible();
  expect(bulkBodies.at(-1)).toMatchObject({
    season: 1,
    aired_only: true,
    watched: false,
  });

  const downloadedSeason = page.getByRole("button", {
    name: "Mark season downloaded",
    exact: true,
  });
  await downloadedSeason.click();
  await expect(
    page.getByRole("button", { name: "Unmark season downloaded", exact: true }),
  ).toBeVisible();
  expect(bulkBodies.at(-1)).toMatchObject({
    season: 1,
    aired_only: true,
    downloaded: true,
  });

  await page
    .getByRole("button", { name: "Unmark season downloaded", exact: true })
    .click();
  await expect(downloadedSeason).toBeVisible();
  expect(bulkBodies.at(-1)).toMatchObject({
    season: 1,
    aired_only: true,
    downloaded: false,
  });
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
  const initialSearchSettings = await (
    await page.request.get("/api/settings/search")
  ).json();
  const originalTimezone = initialBoot.preferences.timezone;
  const userTimezone =
    deploymentSettings.timezone === "America/New_York"
      ? "Europe/Stockholm"
      : "America/New_York";
  await page.request.patch("/api/preferences", {
    headers,
    data: { timezone: userTimezone, time_format: "24h" },
  });
  await page.goto("/admin/configuration/delivery");
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
    .getByRole("button", { name: "Save changes", exact: true })
    .click();
  const savedNotice = page.getByRole("status");
  await expect(savedNotice).toContainText("Notification settings saved");
  await savedNotice.getByRole("button", { name: "Dismiss notification" }).click();
  await expect(savedNotice).toHaveClass(/is-leaving/);
  await expect(savedNotice).toHaveCount(0, { timeout: 1000 });

  await page.getByRole("switch", { name: "Enable all notifications" }).check();
  await expect(
    page.getByRole("switch", { name: "Enable all notifications" }),
  ).toBeChecked();
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  const enabledNotice = page.getByRole("status");
  await expect(enabledNotice).toContainText("Notifications enabled");
  await expect(enabledNotice).toHaveCount(0, { timeout: 3000 });
  await page.reload();
  await expect(
    page.getByRole("switch", { name: "Enable all notifications" }),
  ).toBeChecked();
  await expect(page.getByLabel("Webhook URL", { exact: true })).toHaveValue("");
  await expect(page.getByLabel("Webhook URL", { exact: true })).toHaveAttribute(
    "placeholder",
    "Saved - leave blank to keep it",
  );
  await page
    .getByRole("switch", { name: "Enable all notifications" })
    .uncheck();
  await expect(
    page.getByRole("switch", { name: "Enable all notifications" }),
  ).not.toBeChecked();
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("status")).toContainText(
    "All notifications disabled",
  );
  await page
    .getByRole("link", { name: "Torrent search", exact: true })
    .last()
    .click();
  await expect(page).toHaveURL(/\/admin\/configuration\/integrations\/search$/);
  await page
    .getByLabel("Jackett base URL", { exact: true })
    .fill("http://127.0.0.1:1");
  await page
    .getByRole("button", { name: "Save changes", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Jackett settings saved");
  await page
    .getByRole("checkbox", { name: "Enable torrent search" })
    .uncheck();
  await expect(page.getByRole("status")).toContainText("Torrent search disabled");
  await page.reload();
  const key = page.getByLabel("API key", { exact: true });
  await expect(key).toHaveValue("");
  await expect(key).toHaveAttribute("placeholder", "Saved - leave blank to keep it");
  await expect(key).toHaveAttribute("type", "text");
  await expect(key).toHaveAttribute("autocomplete", "off");
  const disabledSearchSettings = await (
    await page.request.get("/api/settings/search")
  ).json();
  const restoreSearch = await page.request.put("/api/settings/search", {
    headers,
    data: {
      data: initialSearchSettings.data,
      revision: disabledSearchSettings.revision,
    },
  });
  expect(restoreSearch.ok()).toBe(true);
  await page
    .getByRole("link", { name: "Schedules", exact: true })
    .click();
  const schedule = page.locator(".schedule-editor").filter({
    has: page.getByRole("heading", { name: "Maintenance", exact: true }),
  });
  const maintenanceStored = (
    await (await page.request.get("/api/settings/scheduling")).json()
  ).find((job: any) => job.key === "maintenance");
  await schedule
    .getByLabel("Cron expression")
    .fill("15 8 * * 1-5");
  await schedule
    .getByRole("checkbox", { name: "Run automatically" })
    .setChecked(!Boolean(maintenanceStored.enabled));
  await page
    .locator(".unsaved-changes-bar.has-unsaved-changes")
    .getByRole("button", { name: "Save changes", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    maintenanceStored.enabled ? "Maintenance disabled" : "Maintenance enabled",
  );
  await page.reload();
  await expect(
    schedule.getByLabel("Cron expression"),
  ).toHaveValue("15 8 * * 1-5");
  await expect(
    schedule.getByRole("checkbox", { name: "Run automatically" }),
  ).toBeChecked({ checked: !Boolean(maintenanceStored.enabled) });
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
  await page.goto("/admin/operations/jobs");
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
  await page
    .getByRole("button", { name: "Exclude the selected result" })
    .click();
  await expect
    .poll(async () => {
      const boot = await (await page.request.get("/api/bootstrap")).json();
      return {
        job: boot.preferences.job_type_filter,
        status: boot.preferences.job_status_filter,
        statusNot: boot.preferences.job_status_filter_not,
      };
    })
    .toEqual({ job: "backup", status: "failed", statusNot: true });
  await page.reload();
  await expect(
    page.getByRole("combobox", { name: "Filter history by job" }),
  ).toHaveValue("backup");
  await expect(
    page.getByRole("combobox", { name: "Filter history by result" }),
  ).toHaveValue("failed");
  await expect(
    page.getByRole("button", { name: "Exclude the selected result" }),
  ).toHaveAttribute("aria-pressed", "true");
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
  await page.goto("/admin/configuration/schedules");
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
      job_status_filter_not: false,
      timezone: originalTimezone,
    },
  });
  expect(errors).toEqual([]);
});
