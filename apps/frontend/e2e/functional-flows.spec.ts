import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("prevents starting a simulation without active hardware", async ({
  page,
}) => {
  await mockBackend(page, { gateways: "inactive" });
  await page.goto("/simulation/dashboard");

  await expect(
    page.getByRole("button", { name: "Start simulation" }),
  ).toBeDisabled();
  await expect(
    page.getByText("Add at least one active device or gateway before starting a simulation."),
  ).toBeVisible();
});

test("starts a simulation with a single active gateway and no devices", async ({
  page,
}) => {
  await mockBackend(page, { gateways: "active" });
  await page.goto("/simulation/dashboard");

  const start = page.getByRole("button", { name: "Start simulation" });
  await expect(start).toBeEnabled();
  await start.click();
  await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
});

test("saves the Gateway Bridge endpoint from the configuration form", async ({
  page,
}) => {
  await mockBackend(page);
  await page.goto("/hardware/gateway-bridge");

  await expect(page.getByText("Gateway Bridge", { exact: true }).first()).toBeVisible();
  await page.getByText("Enable the Semtech UDP bridge", { exact: true }).click();
  await page.getByLabel("Gateway Bridge's address").fill("127.0.0.1");
  await page.getByLabel("Gateway Bridge's Port").fill("1701");
  await page.getByRole("button", { name: "Save settings" }).click();

  await expect(page.getByText("Enabled", { exact: true })).toBeVisible();
  await expect(page.getByText("127.0.0.1:1701", { exact: true })).toBeVisible();
});
