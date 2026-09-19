import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("shared page headers align and scrollbars do not move content", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  for (const width of [2400, 1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    let left: number | undefined;
    for (const path of [
      "/account",
      "/admin/access/profiles",
      "/admin/operations/jobs",
      "/admin/operations/statistics",
      "/admin/operations/logs",
    ]) {
      await page.goto(path);
      const heading = page.locator(".page > .page-heading").first();
      await expect(heading).toBeVisible();
      const before = (await heading.boundingBox())!;
      if (left !== undefined) expect(Math.abs(before.x - left)).toBeLessThan(1);
      left = before.x;
      await page.evaluate(() => {
        document.body.style.minHeight = "3000px";
      });
      const after = (await heading.boundingBox())!;
      expect(after.x).toBe(before.x);
      expect(after.width).toBe(before.width);
      await page.evaluate(() => {
        document.body.style.minHeight = "";
      });
    }
  }
});

test("header remains visible while scrolling", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/admin/operations/logs");
  const header = page.locator(".topbar");
  await expect(header).toBeVisible();
  await page.evaluate(() => {
    document.body.style.minHeight = "3000px";
    window.scrollTo(0, 900);
  });
  expect((await header.boundingBox())!.y).toBe(0);
});
