const path = require("path");
const { test, expect } = require("@playwright/test");

// Minimal fake parsed GLL data: one box type referencing one source definition.
// The actual file bytes (example-cl.gll) are still loaded so currentFileBytes is
// set, but parseGLL is overridden so the test never depends on the file's contents.
const fakeGLLResult = JSON.stringify({
  success: true,
  data: {
    header: { format_version: 4 },
    gen_system: { label: "Test Speaker", type: 2 },
    metadata: { product_name: "Test Speaker", display_name: "Test Speaker" },
    database: {
      sub_version: 3,
      box_types: [{ key: "bxTest", label: "Test Box", sources: ["sdTest"] }],
      source_definitions: [{ key: "sdTest", label: "Test Source" }],
    },
  },
});

const fakeFrequencies = [125, 250, 500, 1000, 2000, 4000, 8000];
const fakeLevel = fakeFrequencies.map(() => 90);
const fakePhase = fakeFrequencies.map(() => 0);

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

  // Override WASM functions with deterministic mocks so the test never depends
  // on actual acoustic data from the file.
  await page.evaluate(
    ({ gllResult, freqs, level, phase }) => {
      window.parseGLL = () => gllResult;
      window.computeArrayResponse = () =>
        JSON.stringify({ success: true, frequencies: freqs, level, phase });
      window.computeArrayBalloonAsync = (_bytes, _payload, callback) => {
        // 72 meridian × 37 parallel = 2664 grid points
        const results = Array.from({ length: 72 * 37 }, () => ({
          level: freqs.map(() => 90),
          phase: freqs.map(() => 0),
        }));
        setTimeout(
          () =>
            callback(
              JSON.stringify({
                type: "complete",
                success: true,
                result: { success: true, frequencies: freqs, results },
              }),
            ),
          0,
        );
        return JSON.stringify({ type: "started", success: true });
      };
    },
    { gllResult: fakeGLLResult, freqs: fakeFrequencies, level: fakeLevel, phase: fakePhase },
  );

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
  await page.evaluate(({ gllResult }) => {
    window.__cancelArrayBalloonCalls = 0;
    window.__arrayBalloonStarts = 0;
    window.parseGLL = () => gllResult;
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
  }, { gllResult: fakeGLLResult });

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
