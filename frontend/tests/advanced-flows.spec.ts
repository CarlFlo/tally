import { expect, test } from "@playwright/test";
import { selectProfileByName } from "./navigation";

test("place the first node on an empty canvas by dragging or clicking", async ({
  page,
}) => {
  await page.route("**/api/flows", async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { flows: [] } });
      return;
    }
    await route.continue();
  });
  await selectProfileByName(page, "My profile");
  await page.goto("/search/flows");
  const canvas = page.getByLabel("Automation flow canvas");
  await expect(canvas.locator(".react-flow__node")).toHaveCount(0);
  const trigger = page.getByRole("button", { name: "Episode Released" });
  await expect(trigger).toBeEnabled();
  await trigger.dragTo(canvas, { targetPosition: { x: 280, y: 220 } });
  await expect(canvas.locator(".react-flow__node")).toHaveCount(1);
  await expect(canvas.locator(".react-flow__node")).toContainText(
    "Episode Released",
  );
  const canvasBox = await canvas.boundingBox();
  const nodeBox = await canvas.locator(".react-flow__node").boundingBox();
  if (!canvasBox || !nodeBox) throw new Error("Dropped node was not rendered");
  expect(Math.abs(nodeBox.x - canvasBox.x - 280)).toBeLessThan(30);
  expect(Math.abs(nodeBox.y - canvasBox.y - 220)).toBeLessThan(30);
  await expect(page.getByRole("button", { name: "Save", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Build Search Query" }).click();
  await expect(canvas.locator(".react-flow__node")).toHaveCount(2);
});

test("create, edit, reload, and dry-run an advanced flow from an original event", async ({
  page,
}) => {
  await selectProfileByName(page, "My profile");
  await page.goto("/search/runs");
  await expect(
    page.getByRole("link", { name: "Open in Advanced Debugger" }),
  ).toBeVisible();
  await page.getByRole("link", { name: "Open in Advanced Debugger" }).click();
  await expect(page).toHaveURL(/\/search\/flows\?source=/);
  await expect(
    page.getByRole("heading", { name: "Original automation run" }),
  ).toBeVisible();

  await page.getByRole("button", { name: "New flow" }).click();
  const canvas = page.getByLabel("Automation flow canvas");
  await expect(canvas.locator(".react-flow__node")).toHaveCount(6);
  const trigger = canvas.locator('.react-flow__node[data-id="trigger"]');
  const initial = await trigger.boundingBox();
  if (!initial) throw new Error("Trigger node was not rendered");
  await page.mouse.move(initial.x + initial.width / 2, initial.y + 18);
  await page.mouse.down();
  await page.mouse.move(initial.x + initial.width / 2 + 140, initial.y + 88, {
    steps: 8,
  });
  await page.mouse.up();
  const moved = await trigger.boundingBox();
  expect(moved?.x).not.toBe(initial.x);

  await page.getByRole("button", { name: "Stop", exact: true }).click();
  const stop = canvas.locator(".react-flow__node").filter({ hasText: "Stop" });
  await canvas
    .locator(
      '.react-flow__node[data-id="query"] .react-flow__handle[data-handleid="query"]',
    )
    .dragTo(stop.locator('.react-flow__handle[data-handleid="results"]'));
  await expect(page.getByRole("status")).toContainText(
    "These ports cannot be connected",
  );
  await expect(canvas.locator(".react-flow__edge")).toHaveCount(5);
  const source = canvas.locator(
    '.react-flow__node[data-id="search"] .react-flow__handle[data-handleid="no_results"]',
  );
  const target = stop.locator('.react-flow__handle[data-handleid="results"]');
  await source.dragTo(target);
  await expect(canvas.locator(".react-flow__edge")).toHaveCount(6);

  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Flow saved");
  const flowList = await (await page.request.get("/api/flows")).json();
  const saved = flowList.flows.find(
    (flow: { name: string }) => flow.name === "New advanced flow",
  );
  expect(saved).toBeTruthy();
  expect(saved.definition.edges).toHaveLength(6);
  const savedX = saved.definition.nodes.find(
    (node: { id: string }) => node.id === "trigger",
  ).x;
  expect(savedX).not.toBe(0);

  await page.reload();
  await expect(canvas.locator(".react-flow__edge")).toHaveCount(6);
  await expect(
    page.getByRole("combobox", { name: "Historical event" }),
  ).not.toHaveValue("");
  await page.getByRole("button", { name: "Test using this event" }).click();
  await expect(page.getByRole("status")).toContainText("Dry-run test complete");
  await expect(page.getByText("Test run:")).toBeVisible();
  await canvas.locator('.react-flow__node[data-id="filter"]').click();
  await expect(page.getByText(/acceptable/)).toBeVisible();
  await expect(page.locator(".advanced-step")).toContainText(
    "not High confidence: 2",
  );
  await page.getByRole("button", { name: "View recorded graph" }).click();
  await expect(canvas.locator(".react-flow__node")).toHaveCount(7);
  await page.reload();
  await page.locator(".advanced-history button").first().click();
  await expect(
    page.getByRole("button", { name: "Edit current flow" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit current flow" }).click();
  await canvas.locator(".react-flow__node").filter({ hasText: "Stop" }).click();
  await page
    .locator(".advanced-inspector")
    .getByRole("button", { name: "Delete node" })
    .click();
  await expect(canvas.locator(".react-flow__node")).toHaveCount(6);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
