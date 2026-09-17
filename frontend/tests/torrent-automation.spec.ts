import { expect, test, type Page } from "@playwright/test";
import { selectProfileByName } from "./navigation";

async function ensureExampleShow(page: Page) {
  await page.goto("/shows");
  let card = page.locator(".show-card").filter({ hasText: "Example Show" });
  if ((await card.count()) === 0) {
    await page.getByRole("button", { name: "Add show", exact: true }).click();
    await page
      .getByRole("textbox", { name: "Search for a TV show" })
      .fill("Example");
    await expect(
      page.getByRole("heading", { name: "Example Show 2026" }),
    ).toBeVisible();
    await page.locator(".search-show-card").first().hover();
    await page.getByRole("button", { name: "Add", exact: true }).click();
    await expect
      .poll(
        async () => {
          const response = await page.request.get("/api/show-actions");
          if (!response.ok()) return "http";
          const actions = (await response.json()) as Array<{
            name: string;
            status: string;
            followed: boolean | number;
          }>;
          const action = actions.find((item) => item.name === "Example Show");
          return action?.status === "done" && !!action.followed
            ? "done"
            : action?.status || "missing";
        },
        { timeout: 30_000 },
      )
      .toBe("done");
    await page.getByRole("button", { name: "Close dialog" }).click();
    await page.goto("/shows");
    card = page.locator(".show-card").filter({ hasText: "Example Show" });
  }
  await expect(card).toHaveCount(1);
  await card.click();
  await expect(
    page.getByRole("heading", { name: "Example Show." }),
  ).toBeVisible();
  const match = page.url().match(/\/shows\/([^/?#]+)/);
  if (!match) throw new Error("Example Show route did not contain an ID");
  return match[1];
}

test("episode search expands into a strengths and concerns evaluation", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  const showID = await ensureExampleShow(page);

  await page.locator(".episode-row").first().click();
  await page.getByRole("button", { name: "Search torrents" }).click();
  await expect(page).toHaveURL(/\/search\?q=/);
  await expect(
    page.getByRole("textbox", { name: "Torrent search query" }),
  ).toHaveValue("Example Show S01E01");
  await expect(page.locator(".torrent-result")).toHaveCount(2);
  await expect(page.locator(".torrent-result").first().locator(".confidence-high")).toContainText(
    "High · Unverified",
  );

  await page.locator(".torrent-result").first().click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(1);
  await expect(page.getByText("Final evaluation", { exact: true })).toBeVisible();
  await expect(page.getByText("Correct show and episode", { exact: true })).toBeVisible();
  await expect(page.getByText("90 seeders available", { exact: true })).toBeVisible();
  await expect(
    page.getByText("Magnet payload cannot be inspected before download", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("Strengths", { exact: true })).toBeVisible();
  await expect(page.getByText("Concerns", { exact: true })).toBeVisible();

  await page.locator(".torrent-result").nth(1).click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(1);
  await expect(page.locator(".torrent-result").nth(1)).toHaveAttribute("aria-expanded", "true");
  await page.locator(".torrent-result").nth(1).click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(0);

  await page.goto(`/shows/${showID}`);
  await page.getByRole("button", { name: "Show actions: Example Show" }).click();
  await expect(
    page.getByRole("menuitem", {
      name: "Automatic downloads · Use global default",
    }),
  ).toBeVisible();
  await page
    .getByRole("menuitem", {
      name: "Automatic downloads · Never for this show",
    })
    .click();

  const policy = await page.request.get(`/api/torrents/automation/shows/${showID}`);
  expect(policy.ok()).toBe(true);
  expect((await policy.json()).policy).toBe("never");

  const reset = await page.request.put(`/api/torrents/automation/shows/${showID}`, {
    headers: { "X-Tally-CSRF": "1" },
    data: { policy: "default" },
  });
  expect(reset.ok()).toBe(true);
});

test("automation page exposes release filters trust rules and disabled capability guards", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search/automation");
  await expect(
    page.getByRole("link", { name: "Automation", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(
    page.getByText("Enable automatic torrent downloads", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("Release filters", { exact: true })).toBeVisible();
  await expect(page.getByText("Release groups", { exact: true })).toBeVisible();
  await expect(page.getByText("Source trust", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Allowed release groups", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Allowed uploaders", { exact: true })).toBeVisible();
  await expect(
    page.getByLabel("Preferred Jackett providers / indexers", { exact: true }),
  ).toBeVisible();

  const exclude = page.getByLabel("Exclude keywords", { exact: true });
  await expect(exclude).toHaveValue("cam telesync hardsub dubbed");
  await exclude.fill("custom-filter");
  await page.getByRole("button", { name: "Reset filters", exact: true }).click();
  await expect(exclude).toHaveValue("cam telesync hardsub dubbed");

  await expect(
    page.getByText(/Automatic downloads require High confidence and a verified \.torrent payload/),
  ).toBeVisible();
  await expect(
    page.getByText("Magnet-only releases stay manual", { exact: true }),
  ).toBeVisible();

  await page.route("**/api/bootstrap", async (route) => {
    const response = await route.fetch();
    const data = await response.json();
    data.torrent_search_enabled = false;
    data.torrent_downloads_enabled = false;
    await route.fulfill({ response, json: data });
  });
  await page.reload();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(
    page.locator(".sidebar").getByRole("link", { name: "Torrent search" }),
  ).toHaveCount(0);
  await expect(
    page.locator(".sidebar").getByRole("link", { name: "Downloads" }),
  ).toHaveCount(0);
  await page.goto("/search");
  await expect(page).toHaveURL(/\/calendar$/);
  await page.unroute("**/api/bootstrap");
});

test("torrent search tabs remain responsive under rapid repeated navigation", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search");

  for (let index = 0; index < 6; index++) {
    await page.getByRole("link", { name: "Automation", exact: true }).click();
    await expect(page).toHaveURL(/\/search\/automation$/);
    await expect(page.getByLabel("Minimum seeders", { exact: true })).toBeEnabled();

    await page.getByRole("link", { name: "Previous Runs", exact: true }).click();
    await expect(page).toHaveURL(/\/search\/runs$/);
    await expect(page.getByRole("link", { name: "Search", exact: true })).toBeEnabled();

    await page.getByRole("link", { name: "Search", exact: true }).click();
    await expect(page).toHaveURL(/\/search$/);
    const input = page.getByRole("textbox", { name: "Torrent search query" });
    await input.fill(`navigation-${index}`);
    await expect(input).toHaveValue(`navigation-${index}`);
  }
});

test("previous runs explains verified decisions, accepts bad feedback and stays responsive", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  let markedBad = false;
  const run = () => ({
    id: "run-browser",
    show_id: "show-browser",
    episode_id: "episode-browser",
    show_name: "Example Show",
    season: 1,
    episode: 2,
    query: "Example Show S01E02",
    status: "downloaded",
    confidence: "high",
    verification: "verified",
    selected_name: "Example.Show.S01E02.1080p.WEB-DL.mkv",
    selected_infohash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    settings_snapshot: {
      show_policy: "default",
      automation: {
        min_seeders: 5,
        preferred_quality: "1080p",
        release_delay_minutes: 20,
        include_keywords: "",
        exclude_keywords: "cam telesync hardsub dubbed",
        preferred_groups: ["FLUX"],
        preferred_providers: ["Fixture HD"],
      },
    },
    decision_log: [
      {
        stage: "search",
        status: "success",
        summary: "Jackett returned candidates.",
        data: { candidate_count: 2 },
        occurred_at: 1_789_666_800,
      },
      {
        stage: "filter",
        status: "success",
        summary: "One candidate remained eligible.",
        data: {
          rejected: 1,
          below_min_seeders: 0,
          keyword_filtered: 0,
          group_filtered: 0,
          uploader_filtered: 0,
          magnet_only: 0,
          previously_bad: 0,
          shortlisted: 1,
          candidates: [
            {
              rank: 1,
              name: "Example.Show.S01E02.1080p.WEB-DL.mkv",
              provider: "Fixture HD",
              uploader: "fixture-uploader",
              seeders: 90,
              size: 2_147_483_648,
              confidence: "high",
            },
          ],
        },
        occurred_at: 1_789_666_801,
      },
      {
        stage: "verification",
        status: "success",
        summary: "Torrent contents verified.",
        data: {
          name: "Example.Show.S01E02.1080p.WEB-DL.mkv",
          payload: {
            video_files: 1,
            subtitle_files: 1,
            executable_files: 0,
            total_size: 2_147_483_648,
            files: [
              {
                path: "Example.Show.S01E02.1080p.WEB-DL.mkv",
                size: 2_147_483_648,
              },
            ],
          },
        },
        occurred_at: 1_789_666_802,
      },
      {
        stage: "submission",
        status: "success",
        summary: "Submitted to the torrent client.",
        data: { name: "Example.Show.S01E02.1080p.WEB-DL.mkv" },
        occurred_at: 1_789_666_803,
      },
    ],
    engine_version: "1",
    started_at: 1_789_666_800,
    ended_at: 1_789_666_804,
    duration_ms: 4_000,
    ...(markedBad
      ? {
          feedback: {
            profile_id: "profile-browser",
            reason: "wrong_episode",
            note: "Browser feedback fixture",
            created_at: 1_789_666_900,
          },
        }
      : {}),
  });

  await page.route("**/api/torrents/automation/runs?limit=100", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ runs: [run()] }),
    });
  });
  await page.route("**/api/torrents/automation/runs/run-browser", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(run()),
    });
  });
  await page.route("**/api/torrents/automation/runs/run-browser/bad", async (route) => {
    markedBad = true;
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(run()),
    });
  });

  await page.goto("/search/runs");
  await expect(
    page.getByRole("link", { name: "Previous Runs", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(page.getByRole("heading", { name: "Example Show · S01E02" })).toBeVisible();
  await expect(page.getByText("High · Verified", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Verification" })).toBeVisible();
  await page.getByText("Files (1)", { exact: true }).click();
  await expect(
    page.getByText("Example.Show.S01E02.1080p.WEB-DL.mkv", { exact: true }).last(),
  ).toBeVisible();

  await page.getByRole("button", { name: "Mark as bad", exact: true }).click();
  await page.getByText("Note (optional)", { exact: true }).locator("..").getByRole("textbox").fill(
    "Browser feedback fixture",
  );
  await page.getByRole("button", { name: "Mark as bad", exact: true }).click();
  await expect(page.getByText("Marked bad", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("Browser feedback fixture", { exact: true })).toBeVisible();

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".torrent-run-layout")).toBeVisible();
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
  ).toBe(true);
});
