import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

let downloaderSnapshot: any;

const fixtureAPIKey = "qbt_0123456789abcdefghijklmnopqr";

test.afterEach(async ({ page }) => {
  if (!downloaderSnapshot) return;
  const currentResponse = await page.request.get("/api/downloader");
  expect(currentResponse.ok()).toBe(true);
  const current = await currentResponse.json();
  const fields = { ...downloaderSnapshot.settings.fields };
  if (downloaderSnapshot.settings.secrets_configured?.api_key) {
    fields.api_key = fixtureAPIKey;
  }
  const restore = await page.request.put("/api/downloader", {
    headers: { "X-Tally-CSRF": "1" },
    data: {
      adapter: downloaderSnapshot.settings.adapter,
      fields,
      revision: current.settings.revision,
    },
  });
  expect(restore.ok()).toBe(true);
  downloaderSnapshot = undefined;
});

test("configure, test, save and use a shared torrent client with redacted API key controls and protected general APIs", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await selectProfileByName(page, "My profile");
  const initial = await (await page.request.get("/api/downloader")).json();
  downloaderSnapshot = initial;
  const clientURL = initial.settings.fields.url;
  await page.goto("/admin/configuration/integrations/downloader");
  const card = page.locator(".client-settings");
  const featureToggle = page.locator(".feature-toggle-setting");
  await expect(
    featureToggle.getByRole("checkbox", { name: "Enable torrent downloads" }),
  ).toBeChecked();
  await expect(
    card.getByRole("checkbox", { name: "Enable torrent downloads" }),
  ).toHaveCount(0);
  await expect(
    card.getByRole("combobox", { name: "Torrent client", exact: true }),
  ).toHaveValue("qbittorrent");
  await expect(
    page.getByRole("button", { name: "Sign out", exact: true }),
  ).toHaveCount(0);
  await card
    .getByRole("combobox", { name: "Torrent client", exact: true })
    .selectOption("");
  await card.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Torrent client disabled",
  );
  await page.reload();
  await expect(
    card.getByRole("combobox", { name: "Torrent client", exact: true }),
  ).toHaveValue("");
  await expect(
    card.getByRole("button", { name: "Test connection" }),
  ).toBeDisabled();
  await card
    .getByRole("combobox", { name: "Torrent client", exact: true })
    .selectOption("qbittorrent");
  await card
    .getByRole("textbox", { name: "Web UI URL", exact: true })
    .fill(clientURL);
  await expect(card.getByLabel("Username", { exact: true })).toHaveCount(0);
  await expect(card.getByLabel("Password", { exact: true })).toHaveCount(0);
  await card
    .getByLabel("API key", { exact: true })
    .fill("qbt_aaaaaaaaaaaaaaaaaaaaaaaaaaaa");
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("alert")).toContainText("external service request failed");
  await card
    .getByLabel("API key", { exact: true })
    .fill(fixtureAPIKey);
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("status")).toContainText(
    "Authentication and API access verified",
  );
  const unsaved = await (await page.request.get("/api/downloader")).json();
  expect(unsaved.settings.adapter).toBe("");
  await card.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status")).toContainText("Torrent client saved");
  await page.reload();
  await expect(
    card.getByRole("textbox", { name: "Web UI URL", exact: true }),
  ).toHaveValue(clientURL);
  await expect(card.getByLabel("API key", { exact: true })).toHaveValue("");
  await expect(card.getByLabel("API key", { exact: true })).toHaveAttribute(
    "placeholder",
    "Saved - leave blank to keep it",
  );
  await expect(card.getByLabel("API key", { exact: true })).toHaveClass(
    "concealed-secret",
  );
  await expect(card.getByLabel("API key", { exact: true })).toHaveAttribute(
    "type",
    "text",
  );
  await card.getByRole("button", { name: "Show API key", exact: true }).click();
  await expect(card.getByLabel("API key", { exact: true })).toHaveValue("");
  const saved = await (await page.request.get("/api/downloader")).json();
  expect(saved.settings.secrets_configured.api_key).toBe(true);
  expect(saved.settings.fields).not.toHaveProperty("api_key");
  await card
    .getByRole("textbox", { name: "Web UI URL", exact: true })
    .fill(clientURL + "/different");
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("alert")).toContainText("external service request failed");
  await card
    .getByRole("textbox", { name: "Web UI URL", exact: true })
    .fill(clientURL);
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("status")).toContainText(
    "Connected to qBittorrent",
  );
  await page.screenshot({
    path: "../docs/screenshots/torrent-client-settings-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/torrent-client-settings-mobile.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/search");
  await page
    .getByRole("textbox", { name: "Torrent search query" })
    .fill("Example Show");
  await page
    .getByRole("button", { name: "Search torrents", exact: true })
    .click();
  await expect(page.locator(".torrent-result").first()).toBeVisible();
  await page
    .locator(".torrent-result")
    .first()
    .getByRole("button", { name: "Download", exact: true })
    .click();
  await expect(
    page
      .locator(".torrent-result")
      .first()
      .getByRole("button", { name: "Added", exact: true }),
  ).toBeDisabled();
  await page.goto("/admin/configuration/integrations/downloader");
  await card
    .getByRole("button", { name: "Reset", exact: true })
    .click();
  await expect(
    card.getByRole("combobox", { name: "Torrent client", exact: true }),
  ).toHaveValue("");
  await card.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Torrent client disabled",
  );
  expect(
    (await (await page.request.get("/api/downloader")).json()).settings.adapter,
  ).toBe("");
  expect(errors).toEqual([]);
});
