import { test, expect } from "@playwright/test";

const headers = { "X-Tally-CSRF": "1" };

test("backup archives restore and delete from the UI without restarting Tally", async ({ page }) => {
  await page.request.post("/api/profiles/select", {
    headers,
    data: { profile: "user0" },
  });
  await page.goto("/settings");
  await expect(page.locator(".compact-retention strong")).toHaveText("after");

  const keep = page.getByLabel("Automatic backups to keep");
  await keep.fill("3");
  await page.getByRole("button", { name: "Save settings", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Backup settings saved");

  await page.getByRole("button", { name: "Create backup", exact: true }).click();
  const row = page.locator(".backup-row").filter({ hasText: "Manual" }).first();
  await expect(row.getByRole("button", { name: "Restore", exact: true })).toBeVisible({ timeout: 15_000 });
  const filename = await row.locator("strong").innerText();
  const boot = await (await page.request.get("/api/bootstrap")).json();
  await expect(row).toContainText(`Tally ${boot.version}`);

  await keep.fill("4");
  await page.getByRole("button", { name: "Save settings", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Backup settings saved");

  await row.getByRole("button", { name: "Restore", exact: true }).click();
  const restoreDialog = page.getByRole("dialog", { name: "Restore backup" });
  await expect(restoreDialog).toContainText("fully validated");
  await restoreDialog.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(keep).toHaveValue("3", { timeout: 15_000 });

  const restoredRow = page.locator(".backup-row").filter({ hasText: filename });
  await restoredRow.getByRole("button", { name: "Delete", exact: true }).click();
  const deleteDialog = page.getByRole("dialog", { name: "Delete backup" });
  await expect(deleteDialog).toContainText("cannot be recovered");
  await deleteDialog.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(restoredRow).toHaveCount(0);
});
