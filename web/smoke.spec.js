const path = require("path");
const { test, expect } = require("@playwright/test");

test("web demo parses a sample GLL and recalculates visualization outputs", async ({
  page,
}) => {
  const sampleFile = path.join(
    __dirname,
    "..",
    "testdata",
    "gll",
    "example-cl.gll",
  );

  await page.goto("/web/");
  await expect(page.locator("#drop-zone")).toBeVisible();

  await page.waitForFunction(() => typeof window.parseGLL === "function");

  await page.locator("#file-input").setInputFiles(sampleFile);

  await expect(page.locator("#results")).toBeVisible();
  await expect(page.locator("#file-name")).toHaveText("example-cl.gll");

  await page.getByRole("button", { name: "Visualization" }).click();

  const configRows = page.locator("#config-editor-body tr");
  if ((await configRows.count()) === 0) {
    await page.getByRole("button", { name: "Add Element" }).click();
  }

  await expect(configRows.first()).toBeVisible();
  await expect(page.locator(".cfg-box-type").first()).toBeVisible();

  await expect(page.locator("#combined-response-meta")).not.toContainText(
    "Build a configuration above to see combined response",
  );
  await expect(page.locator("#combined-response-meta")).toContainText(
    "Receiver",
  );
  await expect(page.locator("#polar-meta")).toContainText("Frequency");
  await expect(page.locator("#balloon-meta")).toContainText("Frequency");

  await page.locator("#config-clear").click();
  await page.locator("#config-add-element").click();
  await expect(configRows).toHaveCount(1);
  await expect(page.locator("#array-view-placeholder")).toBeHidden();

  await configRows.first().getByRole("button", { name: "X" }).click();
  await expect(configRows).toHaveCount(0);
  await expect(page.locator("#array-view-placeholder")).toBeVisible();
});

test("auto-recalculate cancels the active array computation when a row is removed", async ({
  page,
}) => {
  const sampleFile = path.join(
    __dirname,
    "..",
    "testdata",
    "gll",
    "example-cl.gll",
  );

  await page.goto("/web/");
  await page.waitForFunction(() => typeof window.parseGLL === "function");
  await page.evaluate(() => {
    window.__cancelArrayBalloonCalls = 0;
    window.__arrayBalloonStarts = 0;
    window.computeArrayBalloonAsync = (_data, _payload, callback) => {
      window.__arrayBalloonStarts += 1;
      setTimeout(() => {
        callback(
          JSON.stringify({
            type: "progress",
            completed: 0,
            total: 2664,
          }),
        );
      }, 0);
      return JSON.stringify({ type: "started", success: true });
    };
    window.cancelArrayBalloon = () => {
      window.__cancelArrayBalloonCalls += 1;
      return JSON.stringify({ type: "canceled", success: true, canceled: true });
    };
  });

  await page.locator("#file-input").setInputFiles(sampleFile);
  await expect(page.locator("#results")).toBeVisible();
  await page.getByRole("button", { name: "Visualization" }).click();
  await page.waitForFunction(() => window.__arrayBalloonStarts > 0);

  await page
    .locator("#config-editor-body tr")
    .first()
    .getByRole("button", { name: "X" })
    .click();

  await page.waitForFunction(() => window.__cancelArrayBalloonCalls > 0);
});
