import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

test("torrent search navigation and filters follow the saved Jackett state", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");

  const savedResponse = await page.request.get("/api/settings/search");
  expect(savedResponse.ok()).toBe(true);
  const saved = await savedResponse.json();

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

    await page.route("**/api/torrents/search", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ results, warnings: [] }),
      }),
    );

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
  }
});
