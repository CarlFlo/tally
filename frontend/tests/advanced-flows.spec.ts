import { expect, test, type Page } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("create a vertical preset, test its ordered stages, and reload it", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search/flows");
  await page.getByRole("button", { name: "New preset" }).click();
  await expect(page.getByRole("combobox", { name: "Event" })).toHaveCount(0);
  const chain = page.getByRole("region", { name: "Automation chains" });
  await expect(chain.locator(".advanced-chain-stage")).toHaveCount(5);
  await expect(chain.locator(".react-flow__node")).toHaveCount(0);
  await page.getByRole("textbox", { name: "Chain name" }).fill("Browser preset");
  await chain.getByRole("textbox", { name: "Show title override (optional)" }).fill("Alternate Show");
  await chain.getByRole("textbox", { name: "Words before (optional)" }).fill("WEB-DL");
  await chain.getByRole("spinbutton", { name: "Minimum seeders" }).fill("5");
  await chain.getByRole("spinbutton", { name: "Minimum episode size (MB/min)" }).fill("10");
  await chain.getByRole("spinbutton", { name: "Maximum episode size (MB/min)" }).fill("100");
  await chain.getByRole("textbox", { name: "Allowed release groups" }).fill("FLUX, NTb");
  await chain.getByRole("combobox", { name: "Preferred quality" }).selectOption("2160p");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status")).toContainText("Chain saved");
  await page.getByRole("textbox", { name: "Show name" }).fill("Example Show");
  await page.getByRole("button", { name: "Test", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Dry-run test complete");
  await expect(chain.locator(".advanced-chain-stage").first()).toContainText("WEB-DL Alternate Show");
  await page.reload();
  await page.getByRole("combobox", { name: "Saved chains" }).selectOption({ label: "Browser preset" });
  await expect(chain.locator(".advanced-chain-stage")).toHaveCount(5);
  await expect(chain.getByRole("textbox", { name: "Words before (optional)" })).toHaveValue("WEB-DL");
  await expect(chain.getByRole("textbox", { name: "Allowed release groups" })).toHaveValue("FLUX, NTb");
  const minSize = await chain.getByRole("spinbutton", { name: "Minimum episode size (MB/min)" }).boundingBox();
  const maxSize = await chain.getByRole("spinbutton", { name: "Maximum episode size (MB/min)" }).boundingBox();
  expect(minSize && maxSize && minSize.x < maxSize.x && minSize.y === maxSize.y).toBe(true);
  await expect(chain.getByRole("textbox", { name: "Allowed release groups" })).toHaveCSS("resize", "none");
  await expect(chain.getByRole("textbox", { name: "Allowed uploaders" })).toHaveCSS("overflow-y", "auto");
  const chainBottom = await chain.locator(".advanced-chain-stage").last().boundingBox();
  const historyTop = await page.locator(".advanced-history").boundingBox();
  expect(chainBottom && historyTop && historyTop.y > chainBottom.y + chainBottom.height).toBe(true);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  const list = await (await page.request.get("/api/flows")).json();
  const saved = list.flows.find((flow: { name: string }) => flow.name === "Browser preset");
  expect((await page.request.delete(`/api/flows/${saved.id}`, { headers: { "X-Tally-CSRF": "1" } })).ok()).toBe(true);
});

async function ensureExampleShow(page: Page) {
  await page.goto("/shows");
  let shows = (await (await page.request.get("/api/shows")).json()) as Array<{ id: string; name: string }>;
  let show = shows.find((item) => item.name === "Example Show");
  const created = !show;
  if (!show) {
    await page.getByRole("button", { name: "Add show", exact: true }).click();
    await page.getByRole("textbox", { name: "Search for a TV show" }).fill("Example");
    await expect(page.getByRole("heading", { name: "Example Show 2026" })).toBeVisible();
    await page.locator(".search-show-card").first().hover();
    await page.getByRole("button", { name: "Add", exact: true }).click();
    await expect.poll(async () => {
      const actions = (await (await page.request.get("/api/show-actions")).json()) as Array<{ name: string; status: string; followed: boolean | number }>;
      const action = actions.find((item) => item.name === "Example Show");
      return action?.status === "done" && !!action.followed;
    }, { timeout: 30_000 }).toBe(true);
    shows = (await (await page.request.get("/api/shows")).json()) as Array<{ id: string; name: string }>;
    show = shows.find((item) => item.name === "Example Show");
  }
  if (!show) throw new Error("Example Show is missing");
  return { ...show, created };
}

test("create a preset, assign it to a show, and select it on Automation", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  const show = await ensureExampleShow(page);
  await page.goto("/search/automation");
  const automationHeader = await page.locator(".page-heading").boundingBox();
  const automationTabs = await page.getByRole("navigation", { name: "Torrent search sections" }).boundingBox();
  await page.goto("/search/flows");
  const chainHeader = await page.locator(".page-heading").boundingBox();
  const chainTabs = await page.getByRole("navigation", { name: "Torrent search sections" }).boundingBox();
  expect(chainHeader?.x).toBe(automationHeader?.x);
  expect(chainHeader?.y).toBe(automationHeader?.y);
  expect(chainTabs?.x).toBe(automationTabs?.x);
  expect(chainTabs?.y).toBe(automationTabs?.y);
  await page.getByRole("button", { name: "New preset" }).click();
  await page.getByRole("textbox", { name: "Chain name" }).fill("Browser show chain");
  await page.getByRole("button", { name: "Assign this preset to a show" }).click();
  await page.getByRole("searchbox", { name: "Search shows" }).fill("Example");
  await page.getByRole("option", { name: show.name, exact: true }).click();
  await expect(page.locator(".advanced-chain-event")).toContainText(`Incoming episode for ${show.name}`);
  await page.getByRole("button", { name: "Save and link to show" }).click();
  await expect(page.getByRole("status")).toContainText(`Chain saved and linked to ${show.name}`);
  await page.goto("/search/automation");
  const search = page.getByRole("textbox", { name: "Search My Shows" });
  await search.fill(show.name);
  const row = page.locator(".automation-show-row").filter({ hasText: show.name }).first();
  await row.getByRole("checkbox").check();
  await page.getByRole("button", { name: "Save changes" }).click();
  await page.reload();
  await search.fill(show.name);
  const list = await (await page.request.get("/api/flows")).json();
  const saved = list.flows.find((flow: { name: string }) => flow.name === "Browser show chain");
  await expect(row.getByRole("combobox", { name: "Automation chain" })).toHaveValue(saved.id);
  await expect(row.getByRole("checkbox")).toBeChecked();
  expect((await page.request.put(`/api/torrents/automation/shows/${show.id}`, { headers: { "X-Tally-CSRF": "1" }, data: { enabled: false } })).ok()).toBe(true);
  expect((await page.request.put(`/api/torrents/automation/shows/${show.id}/chain`, { headers: { "X-Tally-CSRF": "1" }, data: { flow_id: "" } })).ok()).toBe(true);
  expect((await page.request.delete(`/api/flows/${saved.id}`, { headers: { "X-Tally-CSRF": "1" } })).ok()).toBe(true);
  if (show.created) expect((await page.request.delete(`/api/shows/${show.id}`, { headers: { "X-Tally-CSRF": "1" } })).ok()).toBe(true);
});

test("edit and reset the live action default chain", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  const original = await (await page.request.get("/api/flows/default-live")).json();
  await page.goto("/search/flows");
  await page.getByRole("combobox", { name: "Saved chains" }).selectOption("default-live");
  const chain = page.getByRole("region", { name: "Automation chains" });
  await chain.getByRole("spinbutton", { name: "Minimum seeders" }).fill("7");
  await page.getByRole("button", { name: "Save changes" }).click();
  await page.reload();
  await page.getByRole("combobox", { name: "Saved chains" }).selectOption("default-live");
  await expect(chain.getByRole("spinbutton", { name: "Minimum seeders" })).toHaveValue("7");
  await page.getByRole("button", { name: "Reset default filters" }).click();
  await page.getByRole("button", { name: "Confirm" }).click();
  await expect(chain.getByRole("spinbutton", { name: "Minimum seeders" })).toHaveValue("5");
  await page.getByRole("combobox", { name: "Saved chains" }).selectOption("default-animated");
  await expect(chain.getByRole("spinbutton", { name: "Minimum episode size (MB/min)" })).toHaveValue("4");
  await expect(chain.getByRole("spinbutton", { name: "Maximum episode size (MB/min)" })).toHaveValue("140");
  const current = await (await page.request.get("/api/flows/default-live")).json();
  expect((await page.request.put("/api/flows/default-live", { headers: { "X-Tally-CSRF": "1" }, data: { ...original, revision: current.revision } })).ok()).toBe(true);
});
