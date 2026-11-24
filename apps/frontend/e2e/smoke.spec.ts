import { expect, test } from "@playwright/test";

test("the application root reaches the simulation dashboard", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveURL(/\/simulation\/dashboard\/?$/, {
    timeout: 15_000,
  });
});

test("the simulation section redirects to its dashboard", async ({ page }) => {
  await page.goto("/simulation");

  await expect(page).toHaveURL(/\/simulation\/dashboard\/?$/, {
    timeout: 15_000,
  });
});
