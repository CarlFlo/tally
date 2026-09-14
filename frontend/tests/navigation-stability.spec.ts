import { expect, test, type Page } from "@playwright/test";
import { openProfileMenu } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

async function clickRoutes(page: Page, paths: string[], rounds: number) {
  for (let round = 0; round < rounds; round++) {
    for (const path of paths) {
      await page.locator(`a[href="${path}"]`).last().click();
      await expect(page).toHaveURL(path);
    }
  }
}

test("rapid view switching stays interactive while live data changes", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });
  await page.request.post("/api/profiles/select", {
    headers,
    data: { profile: "user0" },
  });

  await page.goto("/settings");
  await expect(page.locator(".schedule-editor")).toHaveCount(3);
  const originalDocument = await page.evaluateHandle(() => document);
  const liveChanges = (async () => {
    for (let index = 0; index < 10; index++) {
      await page.request.patch("/api/preferences", {
        headers,
        data: { calendar_view: index % 2 ? "month" : "week" },
      });
      await page.waitForTimeout(170);
    }
  })();

  await clickRoutes(
    page,
    [
      "/settings/torrent",
      "/settings/search",
      "/settings/notifications",
      "/settings/bell",
      "/settings/debug",
      "/settings",
    ],
    5,
  );
  await openProfileMenu(page);
  await page.locator("#profile-menu").getByRole("link", { name: "System", exact: true }).click();
  await expect(page).toHaveURL("/system/jobs");
  await clickRoutes(
    page,
    ["/system/statistics", "/system/logs", "/system/jobs"],
    8,
  );
  await page.locator('.sidebar nav a[href="/calendar"]').click();
  await clickRoutes(page, ["/shows", "/search", "/calendar"], 8);
  await liveChanges;

  await openProfileMenu(page);
  await page.locator("#profile-menu").getByRole("link", { name: "Settings", exact: true }).click();
  expect(await originalDocument.evaluate((original) => original === document)).toBe(true);
  await expect(
    page.getByRole("link", { name: "Torrent client", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("link", { name: "Torrent client", exact: true })
    .click();
  await expect(page).toHaveURL(/\/settings\/torrent$/);
  await expect(
    page.getByRole("button", { name: "Open profile menu" }),
  ).toBeEnabled();
  await expect(page.locator("dialog[open]")).toHaveCount(0);
  await expect(page.locator("[inert]")).toHaveCount(0);
  expect(errors).toEqual([]);
});
