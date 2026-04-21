const { defineConfig } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "./web",
  testMatch: /.*\.spec\.js/,
  timeout: 120000,
  expect: {
    timeout: 30000,
  },
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: "http://127.0.0.1:4173",
    headless: true,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "python3 -m http.server 4173",
    port: 4173,
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
});
