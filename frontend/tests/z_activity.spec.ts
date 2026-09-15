import { test, expect } from "@playwright/test";
const headers = { "X-Tally-CSRF": "1" };
async function selectAdmin(page: any) {
  await page.request.post("/api/profiles/select", {
    headers,
    data: { profile: "user0" },
  });
}

test("show menus clear watched state, preserve downloads, support Shift removal and write searchable logs", async ({
  page,
}) => {
  await selectAdmin(page);
  const response = await page.request.post("/api/shows", {
    headers,
    data: { tvmaze_id: 211 },
  });
  expect(response.ok()).toBe(true);
  const { id } = await response.json();
  await page.goto("/shows");
  const card = page.locator(".show-card-shell").filter({
    has: page.getByRole("heading", { name: "Fixture show 211", exact: true }),
  });
  await card.getByRole("link").click();
  await page
    .getByRole("button", { name: "Mark all aired watched", exact: true })
    .click();
  await expect(page.getByText("All caught up", { exact: true })).toBeVisible();
  const episode = page.locator(".episode-row").first();
  await expect(episode).toHaveClass(/episode-watched/);
  await episode
    .getByRole("button", { name: "Mark downloaded", exact: true })
    .click();
  const actions = page.getByRole("button", {
    name: "Show actions: Fixture show 211",
    exact: true,
  });
  await actions.click();
  await expect(
    page.getByRole("menuitem", { name: "Refresh metadata" }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(actions).toBeFocused();
  await actions.click();
  await page
    .getByRole("menuitem", { name: "Clear watch history", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Confirm", exact: true })
    .click();
  await expect(episode).toHaveClass(/episode-available/);
  await expect(
    episode.getByRole("button", { name: "Mark not downloaded", exact: true }),
  ).toBeVisible();
  await expect(page.getByText("All caught up", { exact: true })).toHaveCount(0);
  await actions.click();
  await page.goBack();
  await expect(page).toHaveURL(/\/shows$/);
  await expect(page.getByRole("menu")).toHaveCount(0);
  await card
    .getByRole("button", { name: "Show actions: Fixture show 211" })
    .click();
  await page
    .getByRole("menuitem", { name: "Remove show", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await card
    .getByRole("button", { name: "Show actions: Fixture show 211" })
    .click();
  await page
    .getByRole("menuitem", { name: "Remove show", exact: true })
    .click({ modifiers: ["Shift"] });
  await expect(card).toHaveCount(0);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.goto("/logs");
  await page
    .getByRole("textbox", { name: "Search logs" })
    .fill("Fixture show 211");
  await page
    .getByRole("combobox", { name: "Filter by action" })
    .selectOption("watch_history_cleared");
  await expect(page.locator(".activity-entry")).toHaveCount(1);
  await expect(page.locator(".activity-entry")).toContainText(
    "Cleared watch history",
  );
  await page.screenshot({
    path: "../docs/screenshots/activity-logs-desktop.png",
    fullPage: true,
  });
  await page
    .getByRole("combobox", { name: "Filter by action" })
    .selectOption("show_removed");
  await expect(page.locator(".activity-entry")).toHaveCount(1);
  await page.request.post("/api/shows", { headers, data: { tvmaze_id: 211 } });
  await page.goto(`/shows/${id}`);
  await page
    .getByRole("button", { name: "Show actions: Fixture show 211" })
    .click();
  await page.getByRole("menuitem", { name: "Refresh metadata" }).click();
  await page.goto("/jobs");
  await page
    .getByRole("combobox", { name: "Filter history by job" })
    .selectOption("all");
  await page
    .getByRole("combobox", { name: "Filter history by result" })
    .selectOption("all");
  await expect(page.locator("table")).toContainText("Fixture show 211");
  await expect(page.locator("table")).not.toContainText(
    "metadata:tvmaze:show:",
  );
  await page.goto(`/shows/${id}`);
  await page.setViewportSize({ width: 390, height: 844 });
  await page
    .getByRole("button", { name: "Show actions: Fixture show 211" })
    .click();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/show-actions-mobile.png",
    fullPage: true,
  });
});

test("login appearance persists on server, admin routes are private, and users can delete their own account", async ({
  page,
}) => {
  await selectAdmin(page);
  const created = await page.request.post("/api/profiles", {
    headers,
    data: { name: "Temporary viewer", avatar: "mint" },
  });
  expect(created.ok()).toBe(true);
  const { id } = await created.json();
  await page.request.post("/api/auth/logout", { headers });
  await page.goto("/login");
  await page.getByRole("button", { name: "Light appearance" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  expect(
    (await (await page.request.get("/api/bootstrap")).json()).browser_theme,
  ).toBe("light");
  await page.getByRole("button", { name: "System appearance" }).click();
  await expect(
    page.getByRole("button", { name: "System appearance" }),
  ).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Dark appearance" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(
    page.getByRole("button", { name: "Dark appearance" }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { name: "Dark appearance" }),
  ).toBeEnabled();
  await page.screenshot({
    path: "../docs/screenshots/login-appearance.png",
    fullPage: true,
  });
  await page.request.post("/api/profiles/select", {
    headers,
    data: { profile: id },
  });
  await page.goto("/calendar");
  for (const name of ["Settings", "Jobs", "Statistics", "Logs"])
    await expect(
      page.locator(".sidebar").getByRole("link", { name, exact: true }),
    ).toHaveCount(0);
  for (const path of ["settings", "jobs", "statistics"]) {
    expect((await page.request.get(`/api/${path}`)).status()).toBe(403);
    await page.goto(`/${path}`);
    await expect(page).toHaveURL(/\/calendar$/);
  }
  await page.goto("/profile");
  await page.getByRole("combobox", { name: "Week starts" }).selectOption("0");
  await page.reload();
  await expect(page.getByRole("combobox", { name: "Week starts" })).toHaveValue(
    "0",
  );
  await page.getByRole("link", { name: "Danger zone", exact: true }).click();
  await page
    .getByRole("button", { name: "Delete my account", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Confirm", exact: true })
    .click();
  await expect(page).toHaveURL(/\/login$/);
  const boot = await (await page.request.get("/api/bootstrap")).json();
  expect(boot.profile).toBeNull();
  expect(boot.profiles.some((p: any) => p.id === id)).toBe(false);
  await selectAdmin(page);
  await page.goto("/profile/danger");
  await expect(
    page.getByText(
      "This is the permanent administrator account. It cannot be deleted.",
    ),
  ).toBeVisible();
});

test("notification forms test Webhook and Discord locally, preserve settings when disabled, and live countdown reaches Available", async ({
  page,
}) => {
  await selectAdmin(page);
  const fixture = await (
    await page.request.get("/__fixture/notifications")
  ).json();
  await page.goto("/settings/notifications");
  await expect(
    page.getByRole("heading", { name: "Notification Services", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("combobox", { name: "Service", exact: true })
    .selectOption("webhook");
  await page.getByLabel("Webhook URL", { exact: true }).fill(fixture.url);
  await page
    .getByLabel("Payload body (JSON)")
    .fill('{"text":"{{message}}","custom":{"event":"{{event}}"}}');
  await page.getByRole("checkbox", { name: "New episode releases" }).check();
  await page.getByLabel("Daily release notification time").fill("18:30");
  await expect(page.getByLabel("Notification timezone")).toHaveCount(0);
  const deployment = await (await page.request.get("/api/settings")).json();
  await expect(page.locator(".notification-timezone-chip")).toHaveText(
    deployment.timezone,
  );
  await page
    .getByRole("button", { name: "Save settings", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Notification settings saved",
  );
  await page
    .getByRole("button", { name: "Test Notification", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Test notification delivered",
  );
  let messages = (
    await (await page.request.get("/__fixture/notifications")).json()
  ).messages;
  expect(messages.at(-1).custom.event).toBe("test");
  await page
    .getByRole("combobox", { name: "Service", exact: true })
    .selectOption("discord");
  await page
    .getByLabel("Discord webhook URL", { exact: true })
    .fill(fixture.url);
  await page
    .getByLabel("Bot display name", { exact: true })
    .fill("Tally fixture");
  await page
    .getByLabel("Custom message prefix", { exact: true })
    .fill("TV update:");
  await page
    .getByRole("button", { name: "Test Notification", exact: true })
    .click();
  await expect
    .poll(
      async () =>
        (
          await (await page.request.get("/__fixture/notifications")).json()
        ).messages.at(-1).username,
    )
    .toBe("Tally fixture");
  await page
    .getByRole("button", { name: "Save settings", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Notification settings saved",
  );
  await page.getByRole("switch", { name: "Enable all notifications" }).check();
  await expect(page.getByRole("status")).toContainText("Notifications enabled");
  await page
    .getByRole("switch", { name: "Enable all notifications" })
    .uncheck();
  await expect(page.getByRole("status")).toContainText(
    "All notifications disabled",
  );
  await page.reload();
  await expect(
    page.getByLabel("Bot display name", { exact: true }),
  ).toHaveValue("Tally fixture");
  await expect(page.getByLabel("Daily release notification time")).toHaveValue(
    "18:30",
  );
  await expect(
    page.getByRole("checkbox", { name: "New episode releases" }),
  ).toBeChecked();
  await page.screenshot({
    path: "../docs/screenshots/notification-services-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/notification-services-mobile.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  const now = new Date();
  await page.clock.install({ time: now });
  await page.request.patch("/api/preferences", {
    headers,
    data: { timezone: "UTC", calendar_view: "month" },
  });
  const stamp = new Date(now.getTime() + 5000).toISOString();
  await page.route("**/api/calendar?*", (route) =>
    route.fulfill({
      json: [
        {
          id: "countdown",
          show_id: "fixture",
          show_name: "Countdown fixture",
          show_image: "",
          name: "Release",
          season: 1,
          number: 1,
          airdate: stamp.slice(0, 10),
          airstamp: stamp,
          watched: 0,
          downloaded: 0,
        },
      ],
    }),
  );
  await page.goto("/calendar");
  const countdown = page
    .locator(".upcoming-card")
    .filter({ hasText: "Countdown fixture" })
    .locator(".release-countdown");
  await expect(countdown).toContainText("in ");
  await page.clock.fastForward(6000);
  await expect(countdown).toHaveText("Available");
  await expect(
    page.getByText("A good story is always worth following."),
  ).toHaveCount(0);
  await page.goto("/search");
  await expect(page.getByText("Always manual", { exact: true })).toHaveCount(0);
});
