import { openProfile, signOut, openProfileMenu, selectProfileByName } from "./navigation";
import { test, expect } from "@playwright/test";

test("profile settings use browser history and signing out stays signed out with one profile", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
  await openProfile(page);
  await expect(page).toHaveURL(/\/profile$/);
  await expect(
    page.getByRole("heading", { name: "My profile." }),
  ).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Display name" })).toHaveValue(
    "My profile",
  );
  await expect(page.locator(".picker-screen")).toHaveCount(0);
  await expect(
    page
      .locator(".sidebar")
      .getByRole("button", { name: "Sign out", exact: true }),
  ).toHaveCount(0);
  await page.goBack();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
  await page.goForward();
  await expect(page).toHaveURL(/\/profile$/);
  await page.getByRole("link", { name: "Security", exact: true }).click();
  await expect(page).toHaveURL(/\/profile\/security$/);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Authentication", exact: true }),
  ).toBeVisible();
  await page.goBack();
  await expect(page).toHaveURL(/\/profile$/);
  await expect(
    page.getByRole("textbox", { name: "Display name" }),
  ).toBeVisible();
  await page.screenshot({
    path: "../docs/screenshots/profile-settings-desktop.png",
    fullPage: true,
  });

  // A failed sign-out must retain the active account and allow retry.
  await page.route("**/api/auth/logout", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "Sign-out test failure; try again" }),
    }),
  );
  await signOut(page);
  await expect(page.getByRole("alert")).toContainText("Sign-out test failure");
  await expect(
    page
      .locator("#profile-menu")
      .getByRole("button", { name: "Sign out", exact: true }),
  ).toBeEnabled();
  await expect(page).toHaveURL(/\/profile$/);
  await page.unroute("**/api/auth/logout");
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();
  await page.reload();
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();
  await page.goBack();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator(".app-shell")).toHaveCount(0);
  await page.getByRole("button", { name: "MY My profile" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
});

test("library, calendar, episode state, profiles, jobs and responsive layout", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await selectProfileByName(page, "My profile");
  await page.goto("/calendar");
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
  await expect(page.locator(".day-cell")).toHaveCount(42);
  await expect(page.locator(".today-number")).toHaveCount(1);
  await page.screenshot({
    path: "../docs/screenshots/calendar-empty-desktop.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Add show", exact: true }).click();
  await page
    .getByRole("textbox", { name: "Search for a TV show" })
    .fill("Example");
  await expect(
    page.getByRole("heading", { name: "Example Show 2026" }),
  ).toBeVisible();
  await page.locator(".search-show-card").first().hover();
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Added", exact: true }),
  ).toBeVisible();
  await expect
    .poll(
      async () => {
        const response = await page.request.get("/api/show-actions");
        if (!response.ok()) return `http-${response.status()}`;
        const actions = (await response.json()) as Array<{
          name: string;
          status: string;
          followed: number | boolean;
        }>;
        const action = actions.find((item) => item.name === "Example Show");
        if (!action) return "missing";
        if (action.status === "failed") return "failed";
        return action.status === "done" && !!action.followed
          ? "done"
          : action.status;
      },
      { timeout: 30_000 },
    )
    .toBe("done");
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.locator(".calendar-episode").first()).toBeVisible();
  await page.locator(".calendar-episode").first().click();
  await expect(page.getByRole("dialog", { name: "Episode details" })).toBeVisible();
  await page.goBack();
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.getByRole("dialog", { name: "Episode details" })).toHaveCount(0);
  await page.locator(".calendar-episode").first().click();
  await page.getByRole("button", { name: "Mark watched", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Watched", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Mark downloaded", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Downloaded", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.reload();
  const completedEpisode = page.locator(".calendar-episode.completed");
  await expect(completedEpisode).toHaveCount(1);
  await expect(completedEpisode.getByLabel("Watched")).toHaveCount(1);
  await expect(completedEpisode.getByLabel("Downloaded")).toHaveCount(1);
  await page.screenshot({
    path: "../docs/screenshots/calendar-desktop.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Agenda", exact: true }).click();
  await expect(page.locator(".agenda-episode").first()).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Agenda", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .getByRole("link", { name: "My shows", exact: true })
    .first()
    .click();
  await page.locator(".show-card").first().click();
  await expect(
    page.getByRole("heading", { name: "Example Show." }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Mark season watched", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Mark season unwatched" }),
  ).toBeVisible();
  await page.locator(".episode-row").first().click();
  await page.getByRole("button", { name: "Search torrents" }).click();
  await expect(page).toHaveURL(/\/search\?q=/);
  await expect(
    page.getByRole("textbox", { name: "Torrent search query" }),
  ).toHaveValue("Example Show S01E01");
  await expect(page.locator(".torrent-result")).toHaveCount(2);
  await page.getByRole("button", { name: "1080p", exact: true }).click();
  await expect(page.locator(".torrent-result")).toHaveCount(1);
  await page
    .locator(".torrent-result")
    .getByRole("button", { name: "Download", exact: true })
    .click();
  await expect(
    page
      .locator(".torrent-result")
      .getByRole("button", { name: "Added", exact: true }),
  ).toBeDisabled();
  await page.screenshot({
    path: "../docs/screenshots/torrent-search-desktop.png",
    fullPage: true,
  });
  await openProfileMenu(page);
  await page.getByRole("link", { name: "Settings", exact: true }).click();
  await page.getByRole("link", { name: "Profiles", exact: true }).click();
  await page.getByRole("button", { name: "New profile" }).click();
  const createDialog = page.getByRole("dialog");
  await createDialog.getByRole("textbox", { name: "Display name" }).fill("Alex");
  await createDialog.getByLabel("New password", { exact: true }).fill("Alex!1234");
  await createDialog.getByLabel("Confirm password", { exact: true }).fill("Alex!1234");
  await createDialog
    .getByRole("button", { name: "Create profile", exact: true })
    .click();
  await expect(page.getByText("Alex", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Switch profile", exact: true }),
  ).toHaveCount(0);
  await openProfile(page);
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await page.getByRole("button", { name: "AL Alex" }).click();
  await expect(
    page.getByRole("heading", { name: "Welcome back, Alex." }),
  ).toBeVisible();
  await page.getByLabel("Password", { exact: true }).fill("Alex!1234");
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(
    page.getByRole("heading", { name: "Your calendar." }),
  ).toBeVisible();
  await expect(page.locator(".calendar-episode")).toHaveCount(0);
  await page.reload();
  await expect(page.locator(".header-profile strong")).toHaveText("Alex");
  await openProfile(page);
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByRole("textbox", { name: "Display name" })).toHaveValue(
    "Alex",
  );
  await signOut(page);
  await page.getByRole("button", { name: "MY My profile" }).click();
  await expect(page.locator(".agenda-episode").first()).toBeVisible();
  await openProfileMenu(page);
  await page.getByRole("link", { name: "System", exact: true }).click();
  await expect(page).toHaveURL(/\/system\/jobs$/);
  await page
    .locator(".job-card")
    .filter({ has: page.getByRole("heading", { name: "Metadata sync" }) })
    .getByRole("button", { name: "Run now" })
    .click();
  await expect(page.locator("table")).toContainText("success");
  await page.getByRole("link", { name: "Statistics", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Statistics." }),
  ).toBeVisible();
  await page.goto("/calendar");
  await page.getByRole("button", { name: "Month", exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Open navigation" }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "Agenda", exact: true }).click();
  await page.screenshot({
    path: "../docs/screenshots/calendar-mobile.png",
    fullPage: true,
  });
  await openProfile(page);
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.locator(".sidebar")).not.toHaveClass(/open/);
  await page.getByRole("button", { name: "Light", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/settings-mobile-light.png",
    fullPage: true,
  });
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  expect(errors).toEqual([]);
});
