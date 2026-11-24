import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("shows a useful empty state when no simulation has been recorded", async ({
  page,
}) => {
  await mockBackend(page, { logs: "empty" });
  await page.goto("/logs");

  await expect(
    page.getByRole("heading", { name: "No simulation logs yet" }),
  ).toBeVisible();
  await expect(
    page.getByText("Start a simulation to create the first run history."),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Open dashboard" })).toBeVisible();
});

test("shows a recoverable error when the logs backend is unavailable", async ({
  page,
}) => {
  await mockBackend(page, { logs: "error" });
  await page.goto("/logs");

  await expect(page.getByText("Unable to load simulation history.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();
});
