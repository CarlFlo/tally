import { test, expect } from "@playwright/test";
import { openProfileMenu, selectProfileByName } from "./navigation";
const headers = { "X-Tally-CSRF": "1" };

test.beforeEach(async ({ page }) => {
  await selectProfileByName(page, "My profile");
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

test("mobile navigation toggles above the header without crowding the breadcrumb", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/calendar");

  const sidebar = page.locator(".sidebar");
  const topbar = page.locator(".topbar");
  const breadcrumb = page.locator(".topbar-breadcrumb");
  const openNavigation = page.getByRole("button", { name: "Open navigation" });

  await expect(openNavigation).toBeVisible();
  await expect(openNavigation).toHaveAttribute("aria-expanded", "false");
  await expect(sidebar).toBeHidden();

  const buttonBox = await openNavigation.boundingBox();
  const breadcrumbBox = await breadcrumb.boundingBox();
  expect(buttonBox).not.toBeNull();
  expect(breadcrumbBox).not.toBeNull();
  expect(breadcrumbBox!.x).toBeGreaterThan(buttonBox!.x + buttonBox!.width);

  await openNavigation.click();

  const closeNavigation = page.getByRole("button", { name: "Close navigation" });
  await expect(closeNavigation).toHaveAttribute("aria-expanded", "true");
  await expect(sidebar).toBeVisible();
  const stacking = await Promise.all([
    sidebar.evaluate((element) => Number(getComputedStyle(element).zIndex)),
    topbar.evaluate((element) => Number(getComputedStyle(element).zIndex)),
    closeNavigation.evaluate((element) => Number(getComputedStyle(element).zIndex)),
  ]);
  expect(stacking[0]).toBeGreaterThan(stacking[1]);
  expect(stacking[2]).toBeGreaterThan(stacking[0]);

  await closeNavigation.click();
  await expect(page.getByRole("button", { name: "Open navigation" })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  await expect(sidebar).toBeHidden();
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
  await page.locator("html").evaluate((element) => {
    element.dataset.profileDeletePageMarker = "stable";
  });
  await profileDelete.click();
  await expect(
    page.getByRole("dialog", { name: "Delete Delete style fixture?" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(page).toHaveURL(/\/admin\/access\/profiles$/);
  await expect(page.locator(".profile-settings-list")).not.toContainText(
    "Delete style fixture",
  );
  await expect(page.locator("html")).toHaveAttribute(
    "data-profile-delete-page-marker",
    "stable",
  );
  await expect(page.locator(".profile-settings-list")).toContainText(
    "admin · Current profile",
  );
  const profileStillExists = (
    await (await page.request.get("/api/bootstrap")).json()
  ).profiles.some((profile: any) => profile.id === profileFixture.id);
  expect(profileStillExists).toBe(false);
  await expect(page.locator(".sidebar").getByRole("link")).toHaveCount(5);
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
  await expect(page).toHaveURL(/\/admin\/operations\/jobs$/);
  await expect(
    page.getByRole("navigation", { name: "Operations sections" }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Statistics", exact: true }).click();
  await expect(page).toHaveURL(/\/admin\/operations\/statistics$/);
  await page.goBack();
  await expect(page).toHaveURL(/\/admin\/operations\/jobs$/);
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
  const stagedSchedule = (
    await (await page.request.get("/api/settings/scheduling")).json()
  ).find((job: any) => job.key === "metadata");
  expect(!!stagedSchedule.enabled).toBe(!!stored.enabled);
  expect(stagedSchedule.schedule).toBe(stored.schedule);
  await expect(
    page.getByRole("button", { name: "Save changes", exact: true }),
  ).toBeVisible();
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

  await page
    .locator(".unsaved-changes-bar.has-unsaved-changes")
    .getByRole("button", { name: "Save changes", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    stored.enabled ? "Metadata sync disabled" : "Metadata sync enabled",
  );
  await page.reload();
  await expect(editor.getByLabel("Cron expression")).toHaveValue("20 * * * *");
  await expect(editor.getByLabel("Run automatically")).toBeChecked({
    checked: !Boolean(stored.enabled),
  });
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
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toBeVisible();
  await page
    .getByRole("checkbox", { name: /Successful backups/ })
    .setChecked(false);
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toHaveCount(0);
  await page
    .getByRole("checkbox", { name: /Successful backups/ })
    .setChecked(true);
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
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
    .getByRole("button", { name: "Save changes", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("Backup settings saved");
  await page.reload();
  await expect(page.getByLabel("Automatic backups to keep")).toHaveValue("3");
  await page
    .getByRole("button", { name: "Create manual backup", exact: true })
    .click();
  const manualBackup = page.locator(".backup-row").filter({ hasText: "Manual" }).first();
  await expect(manualBackup).toBeVisible({ timeout: 15_000 });
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
  await expect(page).toHaveURL(/\/admin\/operations\/logs$/);
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
  await expect(page.getByText("NEXT UP", { exact: true })).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "On the horizon", exact: true }),
  ).toHaveCSS("font-size", "15px");
  const horizonLabel = page.locator(".calendar-rail .horizon-day .tiny-label").first();
  await expect(horizonLabel).toHaveCSS("font-size", "9px");
  await expect(page.locator(".horizon-section .upcoming-card strong").first()).toHaveCSS(
    "font-size",
    "12px",
  );
  await expect(page.locator(".horizon-section .upcoming-card small").first()).toHaveCSS(
    "font-size",
    "10px",
  );
  const horizonScroll = page.locator(".horizon-scroll");
  const idleScrollbarColor = await horizonScroll.evaluate(
    (element) => getComputedStyle(element).scrollbarColor,
  );
  expect(idleScrollbarColor).toMatch(/rgba\(0, 0, 0, 0\)/);
  const scrollbarTransition = await horizonScroll.evaluate(
    (element) => getComputedStyle(element).transitionProperty,
  );
  expect(scrollbarTransition).toContain("--horizon-scrollbar-thumb");
  await page.locator(".horizon-section").hover();
  await expect
    .poll(() =>
      horizonScroll.evaluate((element) => getComputedStyle(element).scrollbarColor),
    )
    .not.toBe(idleScrollbarColor);
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

  // Cached calendar data is available immediately on SPA navigation. The rail
  // must not stretch the grid before the calendar section is measured.
  await page.locator('.sidebar a[href="/shows"]').click();
  await expect(page).toHaveURL(/\/shows$/);
  await page.locator('.sidebar nav a[href="/calendar"]').click();
  await expect(page).toHaveURL(/\/calendar$/);
  const navigatedCalendarBox = await page.locator(".calendar-section").boundingBox();
  const navigatedRailBox = await page.locator(".calendar-rail").boundingBox();
  expect(navigatedCalendarBox).not.toBeNull();
  expect(navigatedRailBox).not.toBeNull();
  expect(navigatedRailBox!.height).toBeLessThanOrEqual(
    navigatedCalendarBox!.height + 2,
  );
  const navigatedHorizonOverflow = await page.locator(".horizon-scroll").evaluate(
    (element) => element.scrollHeight > element.clientHeight,
  );
  expect(navigatedHorizonOverflow).toBe(true);

  await page.locator(".today-cell").getByRole("button", { name: /more$/ }).click();
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
