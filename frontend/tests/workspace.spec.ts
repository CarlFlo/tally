import { test, expect } from "@playwright/test";
import { openProfileMenu } from "./navigation";
const headers = { "X-Tally-CSRF": "1" };

test.beforeEach(async ({ page }) => {
  await page.request.post("/api/profiles/select", {
    headers,
    data: { profile: "user0" },
  });
  await page.request.patch("/api/preferences", {
    headers,
    data: {
      bell_categories: [
        "scheduled_job_failures",
        "backup_failures",
        "episode_releases",
        "provider_api_failures",
        "torrent_client_failures",
      ],
    },
  });
});

test("header navigation, persistent inbox, compact schedules, and downloadable backups", async ({
  page,
}) => {
  await page.goto("/settings/profiles");
  await expect(page.locator(".profile-settings-list")).not.toContainText(
    /user\d+/,
  );
  await expect(page.locator(".profile-settings-list")).toContainText(
    "admin · Current profile",
  );
  await expect(page.locator(".sidebar").getByRole("link")).toHaveCount(4);
  await expect(
    page
      .locator(".topbar")
      .getByRole("button", { name: "Add show", exact: true }),
  ).toHaveCount(0);
  await openProfileMenu(page);
  await page.getByRole("link", { name: "System", exact: true }).click();
  await expect(page).toHaveURL(/\/system\/jobs$/);
  await expect(
    page.getByRole("navigation", { name: "System sections" }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Statistics", exact: true }).click();
  await expect(page).toHaveURL(/\/system\/statistics$/);
  await page.goBack();
  await expect(page).toHaveURL(/\/system\/jobs$/);
  await openProfileMenu(page);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", { name: "Open profile menu" }),
  ).toBeFocused();
  await expect(page.locator("#profile-menu")).toHaveCount(0);

  await page.goto("/settings/scheduling");
  const editor = page
    .locator(".schedule-editor")
    .filter({
      has: page.getByRole("heading", { name: "Metadata sync", exact: true }),
    });
  const stored = (
    await (await page.request.get("/api/settings/scheduling")).json()
  ).find((job: any) => job.key === "metadata");
  await editor.getByLabel("Cron expression").fill("20 * * * *");
  await editor.getByLabel("Run automatically").setChecked(!stored.enabled);
  await expect
    .poll(
      async () =>
        (
          await (await page.request.get("/api/settings/scheduling")).json()
        ).find((job: any) => job.key === "metadata").enabled,
    )
    .toBe(stored.enabled ? 0 : 1);
  expect(
    (await (await page.request.get("/api/settings/scheduling")).json()).find(
      (job: any) => job.key === "metadata",
    ).schedule,
  ).toBe(stored.schedule);
  await expect(editor.getByLabel("Cron expression")).toHaveValue("20 * * * *");
  await editor
    .getByRole("button", { name: "Save schedule", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Schedule saved");
  await page.reload();
  await expect(editor.getByLabel("Cron expression")).toHaveValue("20 * * * *");
  await page.screenshot({
    path: "../docs/screenshots/compact-schedules.png",
    fullPage: true,
  });

  await page.goto("/settings/bell");
  await expect(
    page.getByRole("checkbox", { name: /Successful backups/ }),
  ).not.toBeChecked();
  await expect(
    page.getByRole("checkbox", { name: /New episode releases/ }),
  ).toBeChecked();
  await page
    .getByRole("checkbox", { name: /Successful backups/ })
    .setChecked(true);
  await expect
    .poll(
      async () =>
        (await (await page.request.get("/api/bootstrap")).json()).preferences
          .bell_categories,
    )
    .toContain("backup_successes");

  await page.goto("/settings");
  await expect(
    page.getByRole("link", { name: "Scheduling & backups", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Effective configuration", { exact: true }),
  ).toHaveCount(0);
  await page.getByLabel("Automatic backups to keep").fill("3");
  await page
    .getByRole("button", { name: "Save settings", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Backup settings saved");
  await page.reload();
  await expect(page.getByLabel("Automatic backups to keep")).toHaveValue("3");
  await page
    .getByRole("button", { name: "Create backup", exact: true })
    .click();
  const download = page
    .getByRole("link", { name: "Download", exact: true })
    .first();
  await expect(download).toBeVisible();
  const downloaded = page.waitForEvent("download");
  await download.click();
  const archive = await downloaded;
  expect(archive.suggestedFilename()).toMatch(/\.zip$/);
  expect(await archive.failure()).toBeNull();
  await page.screenshot({
    path: "../docs/screenshots/backup-settings.png",
    fullPage: true,
  });

  const bell = page.getByRole("button", { name: /^Notifications/ });
  await expect(bell).toHaveAccessibleName(/unread/);
  await bell.click();
  const inbox = page.getByRole("region", {
    name: "Notifications",
    exact: true,
  });
  await expect(inbox).toBeVisible();
  await expect(inbox).toContainText("Completed backup job");
  await expect
    .poll(
      async () => (await (await page.request.get("/api/inbox")).json()).unread,
    )
    .toBe(0);
  const message = await inbox.locator(".inbox-entry p").first().innerText();
  await inbox
    .getByRole("button", { name: `Dismiss ${message}`, exact: true })
    .first()
    .click();
  await expect(
    inbox.locator(".inbox-entry p").filter({ hasText: message }),
  ).toHaveCount(0);
  await page.reload();
  await bell.click();
  await expect(
    inbox.locator(".inbox-entry p").filter({ hasText: message }),
  ).toHaveCount(0);
  await page.screenshot({
    path: "../docs/screenshots/notification-inbox.png",
    fullPage: true,
  });
  await expect(inbox).toContainText("You're all caught up.");
  await inbox.getByRole("link", { name: "View all logs" }).click();
  await expect(page).toHaveURL(/\/system\/logs$/);
  await page
    .getByRole("textbox", { name: "Search logs" })
    .fill("metadata schedule");
  await expect(page.locator(".activity-list")).toContainText("20 * * * *");
});

test("calendar combines season releases, groups horizon dates, and expands every show", async ({
  page,
}) => {
  await page.request.patch("/api/preferences", {
    headers,
    data: { calendar_view: "month", timezone: "UTC" },
  });
  const day = new Date().toISOString().slice(0, 10);
  const episode = (show: number, number: number, total: number) => ({
    id: `${show}-${number}`,
    show_id: `group-${show}`,
    show_name: `Release show ${show}`,
    show_image: "",
    name: `Episode ${number}`,
    season: 2,
    number,
    season_episode_count: total,
    type: "regular",
    airdate: day,
    airstamp: day + "T20:00:00Z",
    runtime: 45,
    watched: 0,
    downloaded: 0,
  });
  let episodes = [
    ...Array.from({ length: 8 }, (_, i) => episode(1, i + 1, 8)),
    episode(2, 1, 10),
    episode(2, 2, 10),
    ...Array.from({ length: 4 }, (_, i) => episode(i + 3, 1, 10)),
  ];
  await page.route("**/api/calendar?*", (route) =>
    route.fulfill({ json: episodes }),
  );
  await page.goto("/calendar");
  const today = page.locator(".today-cell");
  await expect(today.locator(".calendar-episode")).toHaveCount(3);
  await expect(today.locator(".calendar-episode").first()).toContainText(
    "Season 2 · Full season",
  );
  await expect(today.locator(".calendar-episode").nth(1)).toContainText(
    "S02E01, S02E02",
  );
  await today.locator(".calendar-episode").first().click();
  await expect(page.getByRole("dialog")).toContainText(
    "Season 2 · Full season",
  );
  await expect(page.getByRole("dialog").locator(".episode-row")).toHaveCount(8);
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Close dialog", exact: true })
    .click();
  await today.getByRole("button", { name: /more$/ }).click();
  await expect(today.locator(".calendar-episode")).toHaveCount(6);
  await expect(page.locator(".horizon-day")).toHaveCount(1);
  await page.screenshot({
    path: "../docs/screenshots/calendar-grouped-releases.png",
    fullPage: true,
  });
  episodes = Array.from({ length: 1005 }, (_, i) => episode(i + 10, 1, 10));
  await page.reload();
  await today.getByRole("button", { name: /more$/ }).click();
  await expect(today.locator(".calendar-episode")).toHaveCount(1005);
});

test("logs refresh in place without replacing an active filter", async ({ page }) => {
  await page.goto("/system/logs");
  const search = page.getByRole("textbox", { name: "Search logs" });
  await search.fill("updated backups settings");
  const saved = await (await page.request.get("/api/settings/backups")).json();
  await page.request.put("/api/settings/backups", {
    headers,
    data: {
      data: { keep: saved.data.keep === 1 ? 2 : 1 },
      revision: saved.revision,
    },
  });
  await expect(page.locator(".activity-list")).toContainText(
    "Updated backups settings",
    { timeout: 7000 },
  );
  await expect(search).toHaveValue("updated backups settings");
});
