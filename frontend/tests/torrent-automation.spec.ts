import { expect, test, type Page } from "@playwright/test";
import { selectProfileByName } from "./navigation";

async function ensureExampleShow(page: Page) {
  await page.goto("/shows");
  async function followedShow() {
    const response = await page.request.get("/api/shows");
    if (!response.ok()) throw new Error(`Could not load followed shows (${response.status()})`);
    const shows = (await response.json()) as Array<{ id: string; name: string }>;
    return shows.find((show) => show.name === "Example Show");
  }

  let show = await followedShow();
  if (!show) {
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
    show = await followedShow();
  }
  if (!show) throw new Error("Example Show was not followed after setup");

  await page.goto(`/shows/${show.id}`);
  await expect(
    page.getByRole("heading", { name: "Example Show." }),
  ).toBeVisible();
  return show.id;
}

async function ensureArchivedShow(page: Page) {
  await page.goto("/shows");
  async function followedShow() {
    const response = await page.request.get("/api/shows");
    if (!response.ok()) throw new Error(`Could not load followed shows (${response.status()})`);
    const shows = (await response.json()) as Array<{ id: string; name: string }>;
    return shows.find((show) => show.name === "Archived Show");
  }

  let show = await followedShow();
  if (!show) {
    await page.getByRole("button", { name: "Add show", exact: true }).click();
    await page
      .getByRole("textbox", { name: "Search for a TV show" })
      .fill("Archived");
    await expect(
      page.getByRole("heading", { name: "Archived Show 2020" }),
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
          const action = actions.find((item) => item.name === "Archived Show");
          return action?.status === "done" && !!action.followed
            ? "done"
            : action?.status || "missing";
        },
        { timeout: 30_000 },
      )
      .toBe("done");
    show = await followedShow();
  }
  if (!show) throw new Error("Archived Show was not followed after setup");
  return show.id;
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
  await expect(page.locator(".torrent-result").first().locator(".confidence-high")).toHaveCount(0);

  await page.locator(".torrent-result").first().click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(1);
  await expect(page.getByText("Final evaluation", { exact: true })).toBeVisible();
  await expect(page.locator(".torrent-result").first().locator(".confidence-high")).toContainText(
    "High · Unverified",
  );
  await expect(page.getByText("Correct show and episode", { exact: true })).toHaveCount(0);
  await expect(page.getByText("90 seeders available", { exact: true })).toBeVisible();
  await expect(page.getByText(/MB\/min is within the configured live-action range/)).toBeVisible();
  await expect(
    page.getByText("Magnet payload cannot be inspected before download", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("Strengths", { exact: true })).toBeVisible();
  await expect(page.getByText("Concerns", { exact: true })).toBeVisible();
  const ratioMeta = page.locator(".torrent-inspection-meta").filter({ hasText: "MB/min" });
  await expect(ratioMeta).toContainText("MB/min");

  await page.locator(".torrent-result").nth(1).click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(1);
  await expect(page.locator(".torrent-result").nth(1)).toHaveAttribute("aria-expanded", "true");
  await page.locator(".torrent-result").nth(1).click();
  await expect(page.locator(".torrent-result-inspection")).toHaveCount(0);

  await page.goto(`/shows/${showID}`);
  const resetEnrollment = await page.request.put(
    `/api/torrents/automation/shows/${showID}`,
    {
      headers: { "X-Tally-CSRF": "1" },
      data: { enabled: false },
    },
  );
  expect(resetEnrollment.ok()).toBe(true);
  await page.reload();

  const enrollmentButton = page.getByRole("button", {
    name: "Automation off",
    exact: true,
  });
  await expect(enrollmentButton).toBeVisible();
  await enrollmentButton.click();
  const policy = await page.request.get(`/api/torrents/automation/shows/${showID}`);
  expect(policy.ok()).toBe(true);
  expect(await policy.json()).toMatchObject({ policy: "auto", enabled: true });
  await expect(
    page.getByRole("button", { name: "Automation on", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Automation on", exact: true }).click();

  await page.getByRole("button", { name: "Show actions: Example Show" }).click();
  await expect(
    page.getByRole("menuitem", { name: /Media type · Auto \(detected / }),
  ).toBeVisible();
  await page.getByRole("menuitem", { name: "Media type · Animated", exact: true }).click();

  const mediaProfile = await page.request.get(
    `/api/torrents/automation/shows/${showID}/media-profile`,
  );
  expect(mediaProfile.ok()).toBe(true);
  expect((await mediaProfile.json()).effective).toBe("animated");

  const resetMedia = await page.request.put(
    `/api/torrents/automation/shows/${showID}/media-profile`,
    {
      headers: { "X-Tally-CSRF": "1" },
      data: { mode: "auto" },
    },
  );
  expect(resetMedia.ok()).toBe(true);
});

test("manual rejected confidence stays reviewable and downloadable", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await ensureExampleShow(page);
  await page.route(/\/api\/torrents\/search\/[^/]+\/evaluation$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        confidence: "rejected",
        verification: "unverified",
        hard_rejections: [{ code: "wrong_episode" }],
      }),
    });
  });

  await page.locator(".episode-row").first().click();
  await page.getByRole("button", { name: "Search torrents" }).click();
  await expect(page).toHaveURL(/\/search\?q=/);
  const result = page.locator(".torrent-result").first();
  await result.click();

  await expect(
    page.getByText(
      "The available metadata has concerns. Review them before deciding whether to download.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(result.getByText("Rejected", { exact: true })).toHaveCount(0);
  await expect(result.locator(".confidence-rejected")).toHaveCount(0);
  await expect(result.getByRole("button", { name: "Download", exact: true })).toBeEnabled();
});

test("automation show list defaults to Running status and search spans all My Shows", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.route("**/api/torrents/automation/shows", async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        shows: [
          {
            id: "running-no-schedule",
            name: "Running Without Schedule",
            status: "Running",
            next_episode: "",
            active: 0,
            automation_enabled: 0,
          },
          {
            id: "ended-future-date",
            name: "Ended With Future Date",
            status: "Ended",
            next_episode: "2099-01-01",
            active: 1,
            automation_enabled: 0,
          },
        ],
      }),
    });
  });

  await page.goto("/search/automation");
  await expect(
    page.locator(".automation-show-row").filter({ hasText: "Running Without Schedule" }),
  ).toBeVisible();
  await expect(
    page.locator(".automation-show-row").filter({ hasText: "Ended With Future Date" }),
  ).toHaveCount(0);

  const search = page.getByRole("textbox", { name: "Search My Shows" });
  await search.fill("Ended With Future Date");
  await expect(
    page.locator(".automation-show-row").filter({ hasText: "Ended With Future Date" }),
  ).toBeVisible();
});

test("automation page exposes release filters trust rules and disabled capability guards", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await ensureExampleShow(page);
  const showID = await ensureArchivedShow(page);
  const resetEnrollment = await page.request.put(
    `/api/torrents/automation/shows/${showID}`,
    {
      headers: { "X-Tally-CSRF": "1" },
      data: { enabled: false },
    },
  );
  expect(resetEnrollment.ok()).toBe(true);
  await page.goto("/search/automation");
  await expect(
    page.getByRole("link", { name: "Automation", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(
    page.getByText("Enable automatic torrent downloads", { exact: true }),
  ).toHaveCount(0);
  await expect(page.getByText("Shows in automation", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toHaveCount(0);

  const minimumSeeders = page.getByLabel("Minimum seeders", { exact: true });
  const savedMinimumSeeders = await minimumSeeders.inputValue();
  await minimumSeeders.fill("4");
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Revert changes", exact: true })).toBeVisible();
  await expect(page.getByText("Unsaved changes", { exact: true })).toBeVisible();
  page.once("dialog", (dialog) => {
    expect(dialog.message()).toContain("unsaved changes");
    void dialog.dismiss();
  });
  await page.getByRole("link", { name: "Previous Runs", exact: true }).click();
  await expect(page).toHaveURL(/\/search\/automation$/);
  await page.getByRole("button", { name: "Revert changes", exact: true }).click();
  await expect(minimumSeeders).toHaveValue(savedMinimumSeeders);
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toHaveCount(0);
  await minimumSeeders.fill("4");

  const exampleRow = page.locator(".automation-show-row").filter({ hasText: "Example Show" });
  await expect(exampleRow).toBeVisible();
  await expect(exampleRow).toContainText("Next episode");

  const archivedRow = page.locator(".automation-show-row").filter({ hasText: "Archived Show" });
  await expect(archivedRow).toHaveCount(0);
  const showSearch = page.getByRole("textbox", { name: "Search My Shows" });
  await showSearch.fill("Archived Show");
  await expect(archivedRow).toBeVisible();
  await expect(archivedRow).toContainText("No upcoming episode");
  const enrollmentToggle = archivedRow.getByRole("checkbox");
  await expect(enrollmentToggle).not.toBeChecked();
  await enrollmentToggle.check();
  await expect(enrollmentToggle).toBeChecked();
  const stagedPolicy = await page.request.get(
    `/api/torrents/automation/shows/${showID}`,
  );
  expect(await stagedPolicy.json()).toMatchObject({ policy: "default", enabled: false });
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("button", { name: "Save changes", exact: true })).toHaveCount(0);
  const enrolledPolicy = await page.request.get(
    `/api/torrents/automation/shows/${showID}`,
  );
  expect(await enrolledPolicy.json()).toMatchObject({ policy: "auto", enabled: true });

  await page.goto(`/shows/${showID}`);
  await expect(
    page.getByRole("button", { name: "Automation on", exact: true }),
  ).toBeVisible();
  await page.goto("/search/automation");
  await showSearch.fill("Archived Show");
  await expect(page.getByText("Release filters", { exact: true })).toBeVisible();
  await expect(page.getByText("Release groups", { exact: true })).toBeVisible();
  await expect(page.getByText("Source trust", { exact: true })).toBeVisible();
  await expect(page.getByText("Episode size", { exact: true })).toBeVisible();
  await expect(page.getByText("Search pacing", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Episode searches per run", { exact: true })).toHaveValue("5");
  await expect(page.getByLabel("Torrent candidates inspected per episode", { exact: true })).toHaveValue("5");
  await expect(page.getByRole("checkbox", { name: "Prioritize newly aired episodes", exact: true })).toBeChecked();
  await expect(page.getByLabel("First retry delay (minutes)", { exact: true })).toHaveValue("30");
  await expect(page.getByLabel("Second retry delay (minutes)", { exact: true })).toHaveValue("120");
  await expect(page.getByLabel("Later retry delay (minutes)", { exact: true })).toHaveValue("360");
  await expect(page.getByLabel("Live-action minimum MB per minute")).toHaveValue("8");
  await expect(page.getByLabel("Live-action maximum MB per minute")).toHaveValue("220");
  await expect(page.getByLabel("Animated minimum MB per minute")).toHaveValue("4");
  await expect(page.getByLabel("Animated maximum MB per minute")).toHaveValue("140");
  expect(
    await page.locator(".torrent-automation-settings").evaluate(
      (element) => getComputedStyle(element).gridTemplateColumns.split(" ").length,
    ),
  ).toBe(2);
  const sizeTrack = page.locator(".dual-range-track").first();
  await expect(sizeTrack).toHaveCSS("left", "8px");
  await expect(sizeTrack).toHaveCSS("right", "8px");
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
    page.getByText(/Automatic downloads require High confidence\. Tally prefers a retrievable \.torrent/),
  ).toBeVisible();
  await expect(
    page.getByText("Magnets are a fallback", { exact: true }),
  ).toBeVisible();

  const disabledBootstrap = await (await page.request.get("/api/bootstrap")).json();
  disabledBootstrap.torrent_search_enabled = false;
  disabledBootstrap.torrent_downloads_enabled = false;
  await page.route("**/api/bootstrap", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(disabledBootstrap),
    });
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

test("experimental torrent automation schedule is confirmed and shared across jobs and settings", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  const headers = { "X-Tally-CSRF": "1" };
  const schedulesResponse = await page.request.get("/api/settings/scheduling");
  expect(schedulesResponse.ok()).toBe(true);
  const schedules = (await schedulesResponse.json()) as Array<{
    key: string;
    schedule: string;
    enabled: boolean | number;
    revision: number;
  }>;
  const saved = schedules.find((job) => job.key === "torrent_automation");
  if (!saved) throw new Error("Torrent automation schedule is missing");

  if (saved.enabled) {
    const disable = await page.request.put("/api/settings/scheduling", {
      headers,
      data: {
        key: saved.key,
        schedule: saved.schedule,
        enabled: false,
        revision: saved.revision,
      },
    });
    expect(disable.ok()).toBe(true);
  }

  try {
    await page.goto("/settings");
    const settingsCard = page
      .locator(".schedule-editor")
      .filter({ hasText: "Torrent automation" });
    const settingsToggle = settingsCard.getByRole("checkbox", {
      name: "Run automatically",
    });
    await expect(settingsToggle).not.toBeChecked();
    await settingsToggle.click();
    await expect(
      page.getByRole("dialog", { name: "Enable experimental job?" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(settingsToggle).not.toBeChecked();

    await page.goto("/system/jobs");
    const jobCard = page.locator(".job-card").filter({ hasText: "Torrent automation" });
    const jobsToggle = jobCard.getByRole("checkbox", {
      name: "Toggle automatic schedule for Torrent automation",
    });
    await expect(jobsToggle).not.toBeChecked();
    await jobsToggle.click();
    await expect(
      page.getByRole("dialog", { name: "Enable experimental job?" }),
    ).toBeVisible();
    await page
      .getByRole("button", {
        name: "Yes, I understand, enable anyway",
        exact: true,
      })
      .click();
    await expect(jobsToggle).toBeChecked();

    await page.goto("/settings");
    const linkedSettingsCard = page
      .locator(".schedule-editor")
      .filter({ hasText: "Torrent automation" });
    const linkedSettingsToggle = linkedSettingsCard.getByRole("checkbox", {
      name: "Run automatically",
    });
    await expect(linkedSettingsToggle).toBeChecked();
    await linkedSettingsToggle.uncheck();
    await expect(linkedSettingsToggle).not.toBeChecked();
    await page.getByRole("button", { name: "Save changes", exact: true }).click();

    await page.goto("/system/jobs");
    await expect(
      page
        .locator(".job-card")
        .filter({ hasText: "Torrent automation" })
        .getByRole("checkbox", {
          name: "Toggle automatic schedule for Torrent automation",
        }),
    ).not.toBeChecked();
  } finally {
    const currentResponse = await page.request.get("/api/settings/scheduling");
    if (currentResponse.ok()) {
      const currentSchedules = (await currentResponse.json()) as Array<{
        key: string;
        schedule: string;
        enabled: boolean | number;
        revision: number;
      }>;
      const current = currentSchedules.find((job) => job.key === saved.key);
      if (
        current &&
        (current.schedule !== saved.schedule || !!current.enabled !== !!saved.enabled)
      ) {
        await page.request.put("/api/settings/scheduling", {
          headers,
          data: {
            key: saved.key,
            schedule: saved.schedule,
            enabled: !!saved.enabled,
            revision: current.revision,
          },
        });
      }
    }
  }
});

test("torrent search tabs stay responsive and cancel abandoned searches", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search");

  for (let index = 0; index < 6; index++) {
    await page.getByRole("link", { name: "Automation", exact: true }).click();
    await expect(page).toHaveURL(/\/search\/automation$/);
    await expect(
      page.getByLabel("Minimum seeders", { exact: true }),
    ).toBeEnabled();

    await page.getByRole("link", { name: "Previous Runs", exact: true }).click();
    await expect(page).toHaveURL(/\/search\/runs$/);
    await expect(
      page.getByRole("link", { name: "Search", exact: true }),
    ).toBeEnabled();

    await page.getByRole("link", { name: "Search", exact: true }).click();
    await expect(page).toHaveURL(/\/search$/);
    const input = page.getByRole("textbox", { name: "Torrent search query" });
    await input.fill(`navigation-${index}`);
    await expect(input).toHaveValue(`navigation-${index}`);
  }

  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/api/torrents/search", async (route) => {
    await gate;
    try {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ results: [], warnings: [] }),
      });
    } catch {
      // Expected when Search unmounts and cancels the abandoned request.
    }
  });

  const input = page.getByRole("textbox", { name: "Torrent search query" });
  await input.fill("Example Show S01E01");
  const pending = page.waitForRequest("**/api/torrents/search");
  await page
    .getByRole("button", { name: "Search torrents", exact: true })
    .click();
  const request = await pending;
  const aborted = page.waitForEvent(
    "requestfailed",
    (failed) => failed === request,
  );

  await page.getByRole("link", { name: "Automation", exact: true }).click();
  await expect(page).toHaveURL(/\/search\/automation$/, { timeout: 1_500 });
  await aborted;
  release();
  await expect(
    page.getByLabel("Minimum seeders", { exact: true }),
  ).toBeEnabled();
  await page.unroute("**/api/torrents/search");
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
    verification: "unverified",
    selected_name: "Example.Show.S01E02.1080p.WEB-DL.mkv",
    selected_infohash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    settings_snapshot: {
      show_policy: "auto",
      show_media_profile: {
        mode: "auto",
        detected: "live",
        effective: "live",
      },
      runtime_minutes: 45,
      automation: {
        min_seeders: 5,
        preferred_quality: "1080p",
        release_delay_minutes: 20,
        include_keywords: "",
        exclude_keywords: "cam telesync hardsub dubbed",
        preferred_groups: ["FLUX"],
        preferred_providers: ["Fixture HD"],
        live_min_mb_per_minute: 8,
        live_max_mb_per_minute: 220,
        animated_min_mb_per_minute: 4,
        animated_max_mb_per_minute: 140,
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
              size_profile: {
                known: true,
                runtime_minutes: 45,
                mb_per_minute: 45.5,
                active_profile: "live",
                active_score: 92,
                live_score: 92,
                animated_score: 64,
                in_active_range: true,
              },
            },
          ],
        },
        occurred_at: 1_789_666_801,
      },
      {
        stage: "inspection",
        status: "success",
        summary: "Candidate payload verified from Jackett torrent metadata.",
        data: {
          name: "Example.Show.S01E02.1080p.WEB-DL.mkv",
          submission_type: "torrent",
          size_profile: {
            known: true,
            runtime_minutes: 45,
            mb_per_minute: 45.5,
            active_profile: "live",
            active_score: 92,
            live_score: 92,
            animated_score: 64,
            in_active_range: true,
          },
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
    engine_version: "4",
    started_at: 1_789_666_800,
    ended_at: 1_789_666_804,
    post_verification: {
      run_id: "run-browser",
      infohash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      status: "verified",
      attempts: 1,
      last_checked_at: 1_789_666_905,
      completed_at: 1_789_666_905,
      assessment: {
        confidence: "high",
        verification: "verified",
        payload: {
          video_files: 1,
          subtitle_files: 0,
          executable_files: 0,
          total_size: 2_147_483_648,
          main_video_size: 2_147_483_648,
          files: [
            {
              path: "Example.Show.S01E02.1080p.WEB-DL.mkv",
              size: 2_147_483_648,
            },
          ],
        },
      },
      size_profile: {
        known: true,
        runtime_minutes: 45,
        mb_per_minute: 45.5,
        active_profile: "live",
        active_score: 92,
        live_score: 92,
        animated_score: 64,
        in_active_range: true,
      },
    },
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
  await expect(page.getByText("High · Unverified", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Post-download verification" })).toBeVisible();
  await expect(page.getByText("Magnet payload verified after submission", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Inspection" })).toBeVisible();
  await expect(page.locator(".size-score-pair .active").first()).toContainText("Live 92");
  await expect(page.locator(".size-score-pair .inactive").first()).toContainText("Animated 64");
  await expect(page.getByText("Auto · detected Live", { exact: true })).toBeVisible();
  const inspectionFiles = page.locator(".torrent-payload-summary details").first();
  await expect(inspectionFiles.locator("summary")).toHaveText("Files (1)");
  await inspectionFiles.locator("summary").click();
  await expect(
    inspectionFiles.getByText("Example.Show.S01E02.1080p.WEB-DL.mkv", { exact: true }),
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
