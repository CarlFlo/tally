import { test, expect } from "@playwright/test";

test("shared page headers align and scrollbars do not move content", async ({
  page,
}) => {
  await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" },
    data: { profile: "user0" },
  });
  for (const width of [2400, 1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    let left: number | undefined;
    for (const path of [
      "/profile",
      "/settings/profiles",
      "/system/jobs",
      "/system/statistics",
      "/system/logs",
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
  await page.request.post("/api/profiles/select", {
    headers: { "X-Tally-CSRF": "1" },
    data: { profile: "user0" },
  });
  await page.goto("/system/logs");
  const header = page.locator(".topbar");
  await expect(header).toBeVisible();
  await page.evaluate(() => {
    document.body.style.minHeight = "3000px";
    window.scrollTo(0, 900);
  });
  expect((await header.boundingBox())!.y).toBe(0);
});
