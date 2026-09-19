import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

let searchSnapshot: any;
let torrentSnapshot: any;

async function restoreSetting(page: any, path: string, snapshot: any) {
  if (!snapshot) return;
  const currentResponse = await page.request.get(path);
  expect(currentResponse.ok()).toBe(true);
  const current = await currentResponse.json();
  if (JSON.stringify(current.data) === JSON.stringify(snapshot.data)) return;
  const restore = await page.request.put(path, {
    headers,
    data: { data: snapshot.data, revision: current.revision },
  });
  expect(restore.ok()).toBe(true);
}

test.beforeEach(async ({ page }) => {
  await selectProfileByName(page, "My profile");
  const searchResponse = await page.request.get("/api/settings/search");
  const torrentResponse = await page.request.get("/api/settings/torrent");
  expect(searchResponse.ok()).toBe(true);
  expect(torrentResponse.ok()).toBe(true);
  searchSnapshot = await searchResponse.json();
  torrentSnapshot = await torrentResponse.json();
});

test.afterEach(async ({ page }) => {
  await restoreSetting(page, "/api/settings/search", searchSnapshot);
  await restoreSetting(page, "/api/settings/torrent", torrentSnapshot);
  searchSnapshot = undefined;
  torrentSnapshot = undefined;
});

test("torrent search navigation and filters follow the saved Jackett state", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");

  const savedResponse = await page.request.get("/api/settings/search");
  expect(savedResponse.ok()).toBe(true);
  const saved = await savedResponse.json();
  const savedTorrentResponse = await page.request.get("/api/settings/torrent");
  expect(savedTorrentResponse.ok()).toBe(true);
  const savedTorrent = await savedTorrentResponse.json();
  let torrentRevision = savedTorrent.revision;
  if (!savedTorrent.data.enabled) {
    const enableDownloads = await page.request.put("/api/settings/torrent", {
      headers,
      data: { data: { enabled: true }, revision: torrentRevision },
    });
    expect(enableDownloads.ok()).toBe(true);
    torrentRevision = (await enableDownloads.json()).revision;
  }

  const enabledData = {
    base_url: "http://127.0.0.1:1",
    api_key: "browser-test-key",
    enabled: true,
  };
  const enabledResponse = await page.request.put("/api/settings/search", {
    headers,
    data: { data: enabledData, revision: saved.revision },
  });
  expect(enabledResponse.ok()).toBe(true);
  let revision = (await enabledResponse.json()).revision;

  try {
    await page.goto("/settings/search");
    const featureToggle = page.locator(".feature-toggle-setting");
    await expect(
      featureToggle.getByRole("checkbox", { name: "Enable torrent search" }),
    ).toBeChecked();
    await expect(
      page
        .locator(".client-settings")
        .getByRole("checkbox", { name: "Enable torrent search" }),
    ).toHaveCount(0);

    const results = [
      {
        id: "result-1080",
        name: "Example S01 1080p WEB-DL x264",
        size: 2_000_000_000,
        seeders: 30,
        leechers: 2,
        provider: "Fixture",
        magnet: "magnet:?xt=urn:btih:" + "a".repeat(40),
        published: "2026-09-16T12:00:00Z",
        download_type: "Magnet",
        sendable: true,
      },
      {
        id: "result-2160",
        name: "Example S01 2160p BluRay x265",
        size: 8_000_000_000,
        seeders: 20,
        leechers: 1,
        provider: "Fixture",
        magnet: "magnet:?xt=urn:btih:" + "b".repeat(40),
        published: "2026-09-16T12:00:00Z",
        download_type: "Magnet",
        sendable: true,
      },
      {
        id: "result-720",
        name: "Example S01 720p WEB-DL x265",
        size: 1_000_000_000,
        seeders: 10,
        leechers: 1,
        provider: "Fixture",
        magnet: "magnet:?xt=urn:btih:" + "c".repeat(40),
        published: "2026-09-16T12:00:00Z",
        download_type: "Magnet",
        sendable: true,
      },
      ...Array.from({ length: 60 }, (_, index) => ({
        id: `bulk-${index}`,
        name: `Bulk ${index} 1080p WEB-DL x264`,
        size: 2_000_000_000,
        seeders: 9,
        leechers: 1,
        provider: "Fixture",
        magnet: "magnet:?xt=urn:btih:" + "d".repeat(40),
        published: "2026-09-16T12:00:00Z",
        download_type: "Magnet",
        sendable: true,
      })),
    ];

    let searchRequests = 0;
    await page.route("**/api/torrents/search", (route) => {
      searchRequests++;
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ results, warnings: [] }),
      });
    });

    let failedSubmission = false;
    await page.route("**/api/torrents/history", (route) => {
      if (route.request().method() !== "GET") return route.continue();
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          searches: [],
          sends: failedSubmission
            ? [
                {
                  name: "Example S01 1080p WEB-DL x264",
                  status: "failed",
                  error: "Fixture downloader failure",
                  created_at: 1_789_563_600,
                },
              ]
            : [],
        }),
      });
    });
    await page.route("**/api/torrents/send", (route) => {
      failedSubmission = true;
      return route.fulfill({
        status: 502,
        contentType: "application/json",
        body: JSON.stringify({ error: "Fixture downloader failure" }),
      });
    });

    await page.goto("/calendar");
    const sidebar = page.locator(".sidebar");
    await expect(
      sidebar.getByRole("link", { name: "Torrent search", exact: true }),
    ).toBeVisible();
    await expect(
      sidebar.getByRole("link", { name: "Downloads", exact: true }),
    ).toBeVisible();

    await sidebar
      .getByRole("link", { name: "Torrent search", exact: true })
      .click();
    await page.getByRole("textbox", { name: "Torrent search query" }).fill("Example");
    await page.getByRole("button", { name: "Search torrents", exact: true }).click();

    await expect(
      page.getByRole("button", { name: "Download", exact: true }).first(),
    ).toBeVisible();

    await page.getByRole("button", { name: "1080p", exact: true }).click();
    await page.getByRole("button", { name: "2160p", exact: true }).click();
    await page.getByRole("button", { name: "x264", exact: true }).click();
    await page.getByRole("button", { name: "x265", exact: true }).click();

    await expect(page.getByText("Example S01 1080p WEB-DL x264", { exact: true })).toBeVisible();
    await expect(page.getByText("Example S01 2160p BluRay x265", { exact: true })).toBeVisible();
    await expect(page.getByText("Example S01 720p WEB-DL x265", { exact: true })).toHaveCount(0);

    const list = page.locator(".torrent-results-list");
    await expect(list).toBeVisible();
    expect(
      await list.evaluate(
        (element) =>
          getComputedStyle(element).overflowY === "auto" &&
          element.scrollHeight > element.clientHeight,
      ),
    ).toBe(true);

    await page.getByLabel("Minimum seeders").fill("15");
    await page.getByLabel("Min size (GB)").fill("1.5");
    await page.getByLabel("Max size (GB)").fill("10");
    await page.getByLabel("Include keywords").fill("Example");
    await page.getByLabel("Exclude keywords").fill("CAM");

    expect(searchRequests).toBe(1);
    await sidebar.getByRole("link", { name: "Calendar", exact: true }).click();
    await expect(page).toHaveURL(/\/calendar$/);
    await sidebar
      .getByRole("link", { name: "Torrent search", exact: true })
      .click();
    await expect(
      page.getByRole("textbox", { name: "Torrent search query" }),
    ).toHaveValue("Example");
    await expect(
      page.getByText("Example S01 1080p WEB-DL x264", { exact: true }),
    ).toBeVisible();
    await expect(page.getByLabel("Minimum seeders")).toHaveValue("15");
    await expect(page.getByLabel("Min size (GB)")).toHaveValue("1.5");
    await expect(page.getByLabel("Max size (GB)")).toHaveValue("10");
    await expect(page.getByLabel("Include keywords")).toHaveValue("Example");
    await expect(page.getByLabel("Exclude keywords")).toHaveValue("CAM");
    await expect(
      page.getByRole("button", { name: "1080p", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    await expect(
      page.getByRole("button", { name: "2160p", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    await expect(
      page.getByRole("button", { name: "x264", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    await expect(
      page.getByRole("button", { name: "x265", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(searchRequests).toBe(1);

    await page
      .locator(".torrent-result")
      .filter({ hasText: "Example S01 1080p WEB-DL x264" })
      .getByRole("button", { name: "Download", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText("Fixture downloader failure");
    const failedRecent = page
      .locator(".history-send")
      .filter({ hasText: "Example S01 1080p WEB-DL x264" });
    await expect(failedRecent).toBeVisible();
    await expect(failedRecent).toContainText("failed");

    const disabledResponse = await page.request.put("/api/settings/search", {
      headers,
      data: {
        data: { ...enabledData, enabled: false },
        revision,
      },
    });
    expect(disabledResponse.ok()).toBe(true);
    revision = (await disabledResponse.json()).revision;

    await expect(
      sidebar.getByRole("link", { name: "Torrent search", exact: true }),
    ).toHaveCount(0);
    await expect(
      sidebar.getByRole("link", { name: "Downloads", exact: true }),
    ).toBeVisible();

    await page.goto("/search");
    await expect(page).toHaveURL(/\/calendar$/);

    const reenabledResponse = await page.request.put("/api/settings/search", {
      headers,
      data: {
        data: enabledData,
        revision,
      },
    });
    expect(reenabledResponse.ok()).toBe(true);
    revision = (await reenabledResponse.json()).revision;
    await expect(
      sidebar.getByRole("link", { name: "Torrent search", exact: true }),
    ).toBeVisible();

    const downloadsDisabled = await page.request.put("/api/settings/torrent", {
      headers,
      data: {
        data: { enabled: false },
        revision: torrentRevision,
      },
    });
    expect(downloadsDisabled.ok()).toBe(true);
    torrentRevision = (await downloadsDisabled.json()).revision;

    await expect(
      sidebar.getByRole("link", { name: "Torrent search", exact: true }),
    ).toBeVisible();
    await expect(
      sidebar.getByRole("link", { name: "Downloads", exact: true }),
    ).toHaveCount(0);

    await page.goto("/search");
    await page.getByRole("textbox", { name: "Torrent search query" }).fill("Example");
    await page.getByRole("button", { name: "Search torrents", exact: true }).click();
    await expect(
      page.getByRole("button", { name: /Copy magnet for Example S01 1080p/ }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Download", exact: true }),
    ).toHaveCount(0);

    await page.goto("/downloads");
    await expect(page).toHaveURL(/\/calendar$/);
    await expect(page.getByRole("heading", { name: "Downloads." })).toHaveCount(0);
  } finally {
    const restore = await page.request.put("/api/settings/search", {
      headers,
      data: { data: saved.data, revision },
    });
    expect(restore.ok()).toBe(true);
    const restoreTorrent = await page.request.put("/api/settings/torrent", {
      headers,
      data: { data: savedTorrent.data, revision: torrentRevision },
    });
    expect(restoreTorrent.ok()).toBe(true);
  }
});


test("downloads layout is present before the torrent client responds", async ({ page }) => {
  await selectProfileByName(page, "My profile");

  const savedResponse = await page.request.get("/api/settings/torrent");
  expect(savedResponse.ok()).toBe(true);
  const saved = await savedResponse.json();
  let revision = saved.revision;
  if (!saved.data.enabled) {
    const enabled = await page.request.put("/api/settings/torrent", {
      headers,
      data: { data: { enabled: true }, revision },
    });
    expect(enabled.ok()).toBe(true);
    revision = (await enabled.json()).revision;
    await page.reload();
  }

  let releaseResponse: (() => void) | undefined;
  const responseGate = new Promise<void>((resolve) => {
    releaseResponse = resolve;
  });
  await page.route("**/api/torrents/downloads", async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    await responseGate;
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        torrents: [],
        stats: { total: 0, active: 0, download_speed: 0, upload_speed: 0 },
      }),
    });
  });

  try {
    await page.goto("/downloads");
    await expect(page.locator(".downloads-stats .stat-card")).toHaveCount(4);
    await expect(page.locator(".downloads-panel")).toBeVisible();
    await expect(page.getByText("Connecting to torrent client…", { exact: true })).toBeVisible();
    await expect(page.getByLabel("Loading")).toHaveCount(0);
    await expect(page.locator(".downloads-stats .stat-card strong")).toHaveText([
      "—",
      "—",
      "—",
      "—",
    ]);

    releaseResponse?.();
    await expect(page.getByText("No Tally downloads", { exact: true })).toBeVisible();
    await expect(page.getByText("Connecting to torrent client…", { exact: true })).toHaveCount(0);
    await expect(page.locator(".downloads-stats .stat-card").first().locator("strong")).toHaveText("0");
  } finally {
    releaseResponse?.();
    if (saved.data.enabled !== true) {
      const restore = await page.request.put("/api/settings/torrent", {
        headers,
        data: { data: saved.data, revision },
      });
      expect(restore.ok()).toBe(true);
    }
  }
});


test("downloads update controls immediately and confirm file deletion", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");

  const savedResponse = await page.request.get("/api/settings/torrent");
  expect(savedResponse.ok()).toBe(true);
  const saved = await savedResponse.json();
  let revision = saved.revision;
  if (!saved.data.enabled) {
    const enabled = await page.request.put("/api/settings/torrent", {
      headers,
      data: { data: { enabled: true }, revision },
    });
    expect(enabled.ok()).toBe(true);
    revision = (await enabled.json()).revision;
    await page.reload();
  }

  const hash = "b".repeat(40);
  let state = "downloading";
  let removed = false;
  let deleteFiles: string | null = null;

  await page.route("**/api/torrents/downloads", (route) => {
    if (route.request().method() !== "GET") return route.continue();
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        torrents: removed
          ? []
          : [
              {
                hash,
                name: "Example download",
                state,
                progress: 0.5,
                size: 1_000,
                downloaded: 500,
                download_speed: state === "downloading" ? 100 : 0,
                upload_speed: 0,
                ratio: 0.2,
                added_on: 1,
                category: "tally",
              },
            ],
        stats: {
          total: removed ? 0 : 1,
          active: !removed && state === "downloading" ? 1 : 0,
          download_speed: !removed && state === "downloading" ? 100 : 0,
          upload_speed: 0,
        },
      }),
    });
  });
  await page.route("**/api/torrents/downloads/**", (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "POST" && url.pathname.endsWith("/stop")) {
      state = "paused";
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ok: true }),
      });
    }
    if (request.method() === "POST" && url.pathname.endsWith("/start")) {
      state = "downloading";
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ok: true }),
      });
    }
    if (request.method() === "DELETE") {
      deleteFiles = url.searchParams.get("delete_files");
      removed = true;
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ok: true }),
      });
    }
    return route.continue();
  });

  try {
    await page.goto("/downloads");
    const row = page.locator(".download-row").filter({ hasText: "Example download" });
    await expect(row).toBeVisible();

    await row.getByRole("button", { name: "Pause", exact: true }).click();
    await expect(
      row.getByRole("button", { name: "Resume", exact: true }),
    ).toBeVisible();

    await row.getByRole("button", { name: "Resume", exact: true }).click();
    await expect(
      row.getByRole("button", { name: "Pause", exact: true }),
    ).toBeVisible();

    await row.getByRole("button", { name: "Remove", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Remove torrent?" });
    await expect(dialog).toBeVisible();
    await expect(
      dialog.getByRole("button", { name: "Remove torrent only", exact: true }),
    ).toBeVisible();
    await expect(
      dialog.getByRole("button", {
        name: "Remove torrent and files",
        exact: true,
      }),
    ).toBeVisible();
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();

    await row.getByRole("button", { name: "Remove", exact: true }).click();
    await page
      .getByRole("dialog", { name: "Remove torrent?" })
      .getByRole("button", {
        name: "Remove torrent and files",
        exact: true,
      })
      .click();
    await expect(page.locator(".download-row")).toHaveCount(0);
    expect(deleteFiles).toBe("true");
  } finally {
    if (saved.data.enabled !== true) {
      const restore = await page.request.put("/api/settings/torrent", {
        headers,
        data: { data: saved.data, revision },
      });
      expect(restore.ok()).toBe(true);
    }
  }
});


test("downloads keep very large speed metrics inside their cards", async ({ page }) => {
  await selectProfileByName(page, "My profile");

  const savedResponse = await page.request.get("/api/settings/torrent");
  expect(savedResponse.ok()).toBe(true);
  const saved = await savedResponse.json();
  let revision = saved.revision;
  if (!saved.data.enabled) {
    const enabled = await page.request.put("/api/settings/torrent", {
      headers,
      data: { data: { enabled: true }, revision },
    });
    expect(enabled.ok()).toBe(true);
    revision = (await enabled.json()).revision;
    await page.reload();
  }

  await page.route("**/api/torrents/downloads", (route) => {
    if (route.request().method() !== "GET") return route.continue();
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        torrents: [],
        stats: {
          total: 0,
          active: 0,
          download_speed: 9_523_372_036_854_776,
          upload_speed: 9_223_372_036_854_775,
        },
      }),
    });
  });

  try {
    await page.goto("/downloads");
    await expect(page.getByText("8.5 PB/s", { exact: true })).toBeVisible();
    await expect(page.getByText("8.2 PB/s", { exact: true })).toBeVisible();
    const cards = page.locator(".downloads-stats .stat-card");
    expect(
      await cards.evaluateAll((items) =>
        items.every((item) => item.scrollWidth <= item.clientWidth),
      ),
    ).toBe(true);
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
  } finally {
    if (saved.data.enabled !== true) {
      const restore = await page.request.put("/api/settings/torrent", {
        headers,
        data: { data: saved.data, revision },
      });
      expect(restore.ok()).toBe(true);
    }
  }
});
