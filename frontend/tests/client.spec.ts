import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("configure, test, save and use a shared torrent client with visible API key controls and protected general APIs", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await selectProfileByName(page, "My profile");
  const initial = await (await page.request.get("/api/downloader")).json();
  const clientURL = initial.settings.fields.url;
  await page.goto("/settings/torrent");
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
  await card.getByRole("button", { name: "Save torrent client" }).click();
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
  await expect(card.getByRole("alert")).toContainText("HTTP 401");
  await card
    .getByLabel("API key", { exact: true })
    .fill("qbt_0123456789abcdefghijklmnopqr");
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("status")).toContainText(
    "Authentication and API access verified",
  );
  const unsaved = await (await page.request.get("/api/downloader")).json();
  expect(unsaved.settings.adapter).toBe("");
  await card.getByRole("button", { name: "Save torrent client" }).click();
  await expect(page.getByRole("status")).toContainText("Torrent client saved");
  await page.reload();
  await expect(
    card.getByRole("textbox", { name: "Web UI URL", exact: true }),
  ).toHaveValue(clientURL);
  await expect(card.getByLabel("API key", { exact: true })).toHaveValue(
    "qbt_0123456789abcdefghijklmnopqr",
  );
  await expect(card.getByLabel("API key", { exact: true })).toHaveClass(
    "concealed-secret",
  );
  await expect(card.getByLabel("API key", { exact: true })).toHaveAttribute(
    "type",
    "text",
  );
  await card.getByRole("button", { name: "Show API key", exact: true }).click();
  const saved = await (await page.request.get("/api/downloader")).json();
  expect(saved.settings.secrets_configured.api_key).toBe(true);
  expect(saved.settings.fields).not.toHaveProperty("api_key");
  await card
    .getByRole("textbox", { name: "Web UI URL", exact: true })
    .fill(clientURL + "/different");
  await card.getByRole("button", { name: "Test connection" }).click();
  await expect(card.getByRole("alert")).toContainText("HTTP 404");
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
  expect(errors).toEqual([]);
});
