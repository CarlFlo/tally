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
  const profileResponse = await page.request.post("/api/profiles", {
    headers,
    data: { name: "Delete style fixture", avatar: "mint" },
  });
  expect(profileResponse.ok()).toBe(true);
  const profileFixture = await profileResponse.json();
  await page.goto("/settings/profiles");
  const profileDelete = page.getByRole("button", {
    name: "Delete Delete style fixture",
    exact: true,
  });
  await expect(profileDelete).toContainText("Delete");
  await expect(profileDelete).toHaveClass(/button/);
  await expect(profileDelete).toHaveClass(/danger/);
  await page.request.delete(`/api/profiles/${profileFixture.id}`, { headers });
  await expect(page.locator(".profile-settings-list")).not.toContainText(
    /user\d+/,
  );
  await expect(page.locator(".profile-settings-list")).toContainText(
    "admin · Current profile",
  );
  await expect(page.locator(".sidebar").getByRole("link")).toHaveCount(4);
  await expect(page.locator(".footer")).toContainText(
    "Tally · Your little TV universe",
  );
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

  await page.request.patch("/api/preferences", {
    headers,
    data: { timezone: "America/New_York", time_format: "24h" },
  });
  await page.goto("/settings/scheduling");
  const editor = page
    .locator(".schedule-editor")
    .filter({
      has: page.getByRole("heading", { name: "Metadata sync", exact: true }),
    });
  const deploymentSettings = await (await page.request.get("/api/settings")).json();
  await expect(page.getByText(
    `Cron schedules use the server timezone: ${deploymentSettings.timezone}.`,
    { exact: true },
  )).toBeVisible();
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

  const fieldsBox = await editor.locator(".schedule-fields").boundingBox();
  const previewBox = await editor.locator(".schedule-preview").boundingBox();
  expect(fieldsBox).not.toBeNull();
  expect(previewBox).not.toBeNull();
  expect(previewBox!.x).toBeGreaterThan(fieldsBox!.x + fieldsBox!.width);
  expect(Math.abs(previewBox!.y - fieldsBox!.y)).toBeLessThan(8);
  const desktopContentWidth = fieldsBox!.width + previewBox!.width;
  expect(previewBox!.width / desktopContentWidth).toBeGreaterThan(0.32);
  expect(previewBox!.width / desktopContentWidth).toBeLessThan(0.38);
  const previewPanel = editor.locator(".schedule-preview");
  await expect(previewPanel.getByText("Cron expression", { exact: true })).toHaveCount(0);
  await expect(previewPanel.getByText("Timezone", { exact: true })).toHaveCount(0);
  await expect(previewPanel).toContainText("Description");
  await expect(previewPanel.getByText("Next 3 runs", { exact: true })).toBeVisible();
  await expect(previewPanel).not.toContainText("UTC");
  const previewResponse = await page.request.post("/api/settings/scheduling/preview", {
    headers,
    data: { schedule: "20 * * * *" },
  });
  const previewData = await previewResponse.json();
  const expectedFirstRun = new Intl.DateTimeFormat("en-GB", {
    timeZone: "America/New_York",
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(previewData.next_runs[0] * 1000));
  await expect(previewPanel.locator("li").first()).toHaveText(expectedFirstRun);

  await page.setViewportSize({ width: 1000, height: 1000 });
  const stackedFieldsBox = await editor.locator(".schedule-fields").boundingBox();
  const stackedPreviewBox = await editor.locator(".schedule-preview").boundingBox();
  expect(stackedFieldsBox).not.toBeNull();
  expect(stackedPreviewBox).not.toBeNull();
  expect(stackedPreviewBox!.y).toBeGreaterThan(stackedFieldsBox!.y + stackedFieldsBox!.height);
  await page.setViewportSize({ width: 1440, height: 1000 });

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
  await page.request.patch("/api/preferences", {
    headers,
    data: { timezone: "UTC", time_format: "24h" },
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
    .getByRole("button", { name: "Create manual backup", exact: true })
    .click();
  const manualBackup = page.locator(".backup-row").filter({ hasText: "Manual" }).first();
  await expect(manualBackup).toBeVisible();
  const backupDelete = manualBackup.getByRole("button", {
    name: "Delete",
    exact: true,
  });
  await expect(backupDelete).toHaveClass(/button/);
  await expect(backupDelete).toHaveClass(/danger/);
  const originalTheme = await page.locator("html").getAttribute("data-theme");
  for (const theme of ["dark", "light"]) {
    await page.locator("html").evaluate((element, value) => {
      element.setAttribute("data-theme", value as string);
    }, theme);
    await backupDelete.hover();
    const hoverStyle = await backupDelete.evaluate((element) => {
      const style = getComputedStyle(element);
      return { background: style.backgroundColor, color: style.color };
    });
    expect(hoverStyle.background).not.toBe("rgba(0, 0, 0, 0)");
    expect(hoverStyle.background).not.toBe(hoverStyle.color);
  }
  await page.locator("html").evaluate((element, value) => {
    if (value) element.setAttribute("data-theme", value as string);
    else element.removeAttribute("data-theme");
  }, originalTheme);
  const download = manualBackup.getByRole("link", { name: "Download", exact: true });
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

  await page.goto("/system/jobs");
  const backupJob = page
    .locator(".job-card")
    .filter({ has: page.getByRole("heading", { name: "Automatic backup", exact: true }) });
  await expect(backupJob).toContainText("Create backups of application data and saved settings.");
  await expect(backupJob.getByText(/Schedule ·/)).toHaveCount(0);
  await backupJob.getByRole("button", { name: "Run now", exact: true }).click();
  await expect
    .poll(async () => {
      const data = await (await page.request.get("/api/backups")).json();
      return data.records.some((record: any) => record.kind === "auto");
    }, { timeout: 15_000 })
    .toBe(true);
  await page.goto("/settings");
  await expect(page.locator(".backup-row").filter({ hasText: "Automatic" }).first()).toBeVisible();

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
  const backupMessages = inbox
    .locator(".inbox-entry p")
    .filter({ hasText: "Completed backup job" });
  const backupMessageCount = await backupMessages.count();
  expect(backupMessageCount).toBeGreaterThanOrEqual(2);
  for (let remaining = backupMessageCount - 1; remaining >= 0; remaining--) {
    await inbox
      .getByRole("button", {
        name: "Dismiss Completed backup job",
        exact: true,
      })
      .first()
      .click();
    await expect(backupMessages).toHaveCount(remaining);
  }
  await page.reload();
  await bell.click();
  await expect(backupMessages).toHaveCount(0);
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
  await expect(page.getByText("shows in your orbit", { exact: true })).toHaveCount(0);
  await expect(page.getByText("episodes this view", { exact: true })).toHaveCount(0);
  await expect(page.getByText("already caught up", { exact: true })).toHaveCount(0);
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
  const horizonLabel = page.locator(".calendar-rail .tiny-label").first();
  await expect(horizonLabel).toHaveCSS("font-size", "8px");
  const horizonScroll = page.locator(".horizon-scroll");
  const idleScrollbarColor = await horizonScroll.evaluate(
    (element) => getComputedStyle(element).scrollbarColor,
  );
  expect(idleScrollbarColor).toContain("transparent");
  await page.locator(".horizon-section").hover();
  const hoverScrollbarColor = await horizonScroll.evaluate(
    (element) => getComputedStyle(element).scrollbarColor,
  );
  expect(hoverScrollbarColor).not.toBe(idleScrollbarColor);
  await page.screenshot({
    path: "../docs/screenshots/calendar-grouped-releases.png",
    fullPage: true,
  });
  episodes = Array.from({ length: 1005 }, (_, i) => episode(i + 10, 1, 10));
  await page.reload();
  const calendarBox = await page.locator(".calendar-section").boundingBox();
  const railBox = await page.locator(".calendar-rail").boundingBox();
  expect(calendarBox).not.toBeNull();
  expect(railBox).not.toBeNull();
  expect(railBox!.height).toBeLessThanOrEqual(calendarBox!.height + 2);
  const horizonOverflow = await page.locator(".horizon-scroll").evaluate(
    (element) => element.scrollHeight > element.clientHeight,
  );
  expect(horizonOverflow).toBe(true);
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
