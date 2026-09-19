import { test, expect } from "@playwright/test";
import { selectProfileByName } from "./navigation";

const headers = { "X-Tally-CSRF": "1" };

test("backup archives restore relationships/preferences and delete without fake failure rows", async ({ page }) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/admin/configuration/backups");
  await expect(page.locator(".compact-retention strong")).toHaveText("after");

  const keep = page.getByLabel("Automatic backups to keep");
  const originalKeep = await keep.inputValue();
  await page
    .getByRole("button", { name: "Create manual backup", exact: true })
    .click();
  const row = page.locator(".backup-row").filter({ hasText: "Manual" }).first();
  await expect(row.getByRole("button", { name: "Restore", exact: true })).toBeVisible({
    timeout: 15_000,
  });
  const filename = await row.locator("strong").innerText();
  const boot = await (await page.request.get("/api/bootstrap")).json();
  await expect(row).toContainText(`Tally ${boot.version}`);
  await expect(page.getByText("Backup could not be completed", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Failed", { exact: true })).toHaveCount(0);

  const changedKeep = originalKeep === "4" ? "5" : "4";
  await keep.fill(changedKeep);
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Backup settings saved");

  await row.getByRole("button", { name: "Restore", exact: true }).click();
  const restoreDialog = page.getByRole("dialog", { name: "Restore backup" });
  await expect(restoreDialog).toContainText("fully validated");
  await restoreDialog.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(keep).toHaveValue(originalKeep, { timeout: 15_000 });

  const restoredRow = page.locator(".backup-row").filter({ hasText: filename });
  await restoredRow.getByRole("button", { name: "Delete", exact: true }).click();
  const deleteDialog = page.getByRole("dialog", { name: "Delete backup" });
  await expect(deleteDialog).toContainText("cannot be recovered");
  await deleteDialog.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(restoredRow).toHaveCount(0);
});
