import { openProfile, signOut, openProfileMenu } from "./navigation";
import { test, expect } from "@playwright/test";

test.use({ baseURL: "http://127.0.0.1:18082" });

test("local sign-in follows browser history and switching requires sign-out", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/calendar");
  await expect(page).toHaveURL(/\/login$/);
  await page.getByRole("button", { name: "M My profile" }).click();
  await expect(page).toHaveURL(/\/login\/user0$/);
  await page
    .getByLabel("Password or PIN", { exact: true })
    .fill("unsent-password");
  await page.goBack();
  await expect(page).toHaveURL(/\/login$/);
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();
  await page.goForward();
  await expect(page).toHaveURL(/\/login\/user0$/);
  await expect(page.getByLabel("Password or PIN", { exact: true })).toHaveValue(
    "",
  );
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Welcome back, My profile." }),
  ).toBeVisible();
  await page.getByLabel("Password or PIN", { exact: true }).fill("1234");
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page).toHaveURL(/\/calendar$/);
  await openProfile(page);
  await expect(page).toHaveURL(/\/profile$/);
  await page.getByRole("link", { name: "Security", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Your sessions" }),
  ).toBeVisible();
  await expect(page.getByText("This browser", { exact: true })).toBeVisible();
  const otherTab = await page.context().newPage();
  await otherTab.goto("/profile");
  await expect(
    otherTab.getByRole("textbox", { name: "Display name" }),
  ).toHaveValue("My profile");
  await signOut(page);
  await expect(page).toHaveURL(/\/login$/);
  await expect(otherTab).toHaveURL(/\/login$/);
  await expect(otherTab.locator(".app-shell")).toHaveCount(0);
  await page.goBack();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator(".app-shell")).toHaveCount(0);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Who's keeping up?" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "A Alex" }).click();
  await page.getByLabel("Password or PIN", { exact: true }).fill("1234");
  await page.getByRole("button", { name: "Enter your space" }).click();
  await expect(page.locator(".header-profile strong")).toHaveText("Alex");
  await expect(otherTab).toHaveURL(/\/calendar$/);
  await expect(otherTab.locator(".header-profile strong")).toHaveText("Alex");
  await page.goto("/login/user0");
  await expect(page).toHaveURL(/\/calendar$/);
  await expect(page.locator(".header-profile strong")).toHaveText("Alex");
  expect(errors).toEqual([]);
  await otherTab.close();
});
