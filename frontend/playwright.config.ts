import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  workers: 1,
  fullyParallel: false,
  timeout: 60_000,
  use: {
    baseURL: "http://127.0.0.1:18080",
    viewport: { width: 1440, height: 1000 },
    colorScheme: "dark",
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command:
        "go test ./internal/api -run TestBrowserServer -count=1 -timeout=10m",
      cwd: "..",
      url: "http://127.0.0.1:18080/readyz",
      reuseExistingServer: false,
      timeout: 180_000,
      stdout: "pipe",
      stderr: "pipe",
      env: { TALLY_BROWSER_TEST: "1" },
    },
    {
      command:
        "go test ./internal/api -run TestBrowserServer -count=1 -timeout=10m",
      cwd: "..",
      url: "http://127.0.0.1:18082/readyz",
      reuseExistingServer: false,
      timeout: 180_000,
      stdout: "pipe",
      stderr: "pipe",
      env: { TALLY_BROWSER_TEST: "1", TALLY_BROWSER_AUTH: "local" },
    },
  ],
});
