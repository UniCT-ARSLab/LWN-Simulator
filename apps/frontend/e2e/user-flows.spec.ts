import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("the user can switch profile from the navigation menu", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");

  await expect(page.getByText("Devices", { exact: true }).first()).toBeVisible();
  await page.getByRole("button", { name: "Select profile" }).click();
  await page.locator(".profile-menu-selector .selected-box").click();
  await page.getByText("Test profile", { exact: true }).click();

  await expect(page.locator(".profile-menu-name")).toHaveText("Test profile");
});

test("the user can open the newest simulation log", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/logs");

  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();
  const newestRun = page.locator('a[href*="run-new"]');
  await expect(newestRun).toBeVisible();
  await newestRun.click();

  await expect(page).toHaveURL(/\/logs\/detail\/?\?runId=run-new$/);
});
