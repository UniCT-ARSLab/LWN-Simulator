import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("loads the logs route without unexpected client errors within the expected budget", async ({
  page,
}) => {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];

  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });

  await mockBackend(page);
  const response = await page.goto("/logs", { waitUntil: "networkidle" });
  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();

  expect(response?.ok()).toBe(true);
  expect(pageErrors).toEqual([]);
  const unexpectedConsoleErrors = consoleErrors.filter(
    (message) => !message.includes("/api/ws"),
  );
  expect(unexpectedConsoleErrors).toEqual([]);

  const domContentLoadedMilliseconds = await page.evaluate(() => {
    const navigation = performance.getEntriesByType(
      "navigation",
    )[0] as PerformanceNavigationTiming | undefined;
    return navigation?.domContentLoadedEventEnd ?? 0;
  });
  expect(domContentLoadedMilliseconds).toBeLessThan(10_000);
});

test("keeps the primary navigation usable after repeated route changes", async ({
  page,
}) => {
  await mockBackend(page);

  for (const route of ["/logs", "/hardware/devices", "/logs"]) {
    await page.goto(route, { waitUntil: "domcontentloaded" });
    await expect(page.getByRole("navigation").first()).toBeVisible();
  }

  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Simulation" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Hardware" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Logs", exact: true })).toBeVisible();
});
