import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

const repositoryDirectory = path.resolve(process.cwd(), "../..");
const externalServer = process.env.PLAYWRIGHT_EXTERNAL_SERVER === "true";
const reuseExport = process.env.PLAYWRIGHT_REUSE_EXPORT === "true";
const crossBrowser = process.env.PLAYWRIGHT_CROSS_BROWSER === "true";
const playwrightPort = process.env.PLAYWRIGHT_PORT ?? "3000";
const playwrightBaseURL = `http://127.0.0.1:${playwrightPort}`;
const chromiumChannel = process.env.PLAYWRIGHT_CHROMIUM_CHANNEL;

const projects = [
  {
    name: "chromium",
    use: {
      ...devices["Desktop Chrome"],
      ...(chromiumChannel ? { channel: chromiumChannel } : {}),
    },
  },
];

if (crossBrowser) {
  projects.push(
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
    },
    {
      name: "webkit",
      use: { ...devices["Desktop Safari"] },
    },
  );
}

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI
    ? [
        ["github"],
        ["html", { outputFolder: "playwright-report", open: "never" }],
      ]
    : "list",
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? playwrightBaseURL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects,
  webServer: externalServer
    ? undefined
    : {
        command: reuseExport
          ? "node apps/frontend/e2e/static-server.mjs"
          : "yarn workspace @lwn-simulator/frontend build && node apps/frontend/e2e/static-server.mjs",
        cwd: repositoryDirectory,
        url: `http://127.0.0.1:${playwrightPort}`,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
});
