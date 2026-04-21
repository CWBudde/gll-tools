const path = require("path");
const { test, expect } = require("@playwright/test");

test("web demo parses a sample GLL and shows visualization empty states", async ({
  page,
}) => {
  const sampleFile = path.join(
    __dirname,
    "..",
    "testdata",
    "gll",
    "example-ls.gll",
  );

  await page.goto("/web/");
  await expect(page.locator("#drop-zone")).toBeVisible();

  await page.waitForFunction(() => typeof window.parseGLL === "function");

  await page.locator("#file-input").setInputFiles(sampleFile);

  await expect(page.locator("#results")).toBeVisible();
  await expect(page.locator("#file-name")).toHaveText("example-ls.gll");

  await page.getByRole("button", { name: "Visualization" }).click();

  const configRows = page.locator("#config-editor-body tr");
  if ((await configRows.count()) === 0) {
    await page.getByRole("button", { name: "Add Element" }).click();
  }

  await expect(configRows.first()).toBeVisible();
  await expect(page.locator(".cfg-box-type").first()).toBeVisible();
  await expect(page.locator("#polar-meta")).toContainText(
    "No polar data available",
  );
  await expect(page.locator("#balloon-meta")).toContainText("No balloon data");
  await expect(page.locator("#combined-response-meta")).toContainText(
    "Build a configuration above to see combined response",
  );
});
