import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test.use({
  hasTouch: true,
  isMobile: true,
  viewport: { height: 844, width: 390 },
});

test("the logs page remains usable on a mobile viewport", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/logs");

  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Simulation" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Hardware" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Logs", exact: true })).toBeVisible();

  const hasHorizontalOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(hasHorizontalOverflow).toBe(false);
});

test("primary controls remain available on a mobile viewport", async ({
  page,
}) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");

  await expect(page.getByRole("button", { name: "Select profile" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Select profile" })).toBeEnabled();
  const themeButton = page.locator(".navbar > .actions > button");
  await expect(themeButton).toBeVisible();
  await expect(page.getByRole("button", { name: "Add Device" })).toBeVisible();
  await expect(page.getByRole("link", { name: /Documentation/ })).toHaveAttribute(
    "target",
    "_blank",
  );
});
