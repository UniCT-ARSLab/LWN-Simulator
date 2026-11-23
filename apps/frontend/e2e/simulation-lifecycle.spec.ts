import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("runs the complete simulation control lifecycle", async ({ page }) => {
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  const startButton = page.getByRole("button", { name: "Start simulation" });
  await expect(startButton).toBeEnabled({ timeout: 15_000 });

  await page.getByLabel("Seed (optional)").fill("42");
  await startButton.click();

  await expect(page.getByText("Running", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Pause" })).toBeVisible();

  await page.getByRole("button", { name: "Pause" }).click();
  await expect(page.getByText("Paused", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Resume" })).toBeVisible();

  await page.getByRole("button", { name: "Resume" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Stop" }).click();
  await expect(page.getByText("Stopped", { exact: true })).toBeVisible();
  await expect(startButton).toBeEnabled();
});

test("locks external bridge simulations to real time", async ({ page }) => {
  await mockBackend(page, { devices: "active", bridgeEnabled: true });
  await page.goto("/simulation/dashboard");

  await expect(page.getByRole("slider")).toBeDisabled();
  await expect(page.getByText("External gateway connections use real time. Simulation speed is fixed at 1x.")).toBeVisible();

  const startRequest = page.waitForRequest((request) =>
    request.method() === "POST" && request.url().includes("/simulation/start"),
  );
  await page.getByRole("button", { name: "Start simulation" }).click();
  const request = await startRequest;
  expect(request.postDataJSON()).toMatchObject({ speed: 1, gatewayBridge: { enabled: true } });
  await expect(page.getByRole("slider")).toBeDisabled();
});

test("unlocks device and gateway fields after stop without a WebSocket notification", async ({
  page,
}) => {
  await page.routeWebSocket("**/api/ws", () => {});
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  const start = page.getByRole("button", { name: "Start simulation" });
  const stop = page.getByRole("button", { name: "Stop", exact: true });
  await start.click();
  await expect(stop).toBeVisible();

  const openEditor = async (section: "devices" | "gateways", name: string) => {
    await page.getByRole("link", { name: "Hardware", exact: true }).click();
    if (section === "gateways") {
      await page.locator('aside a[href^="/hardware/gateways"]').click();
    }
    await page.getByRole("cell", { name, exact: true }).click();
    await page.getByRole("button", { name: "Edit", exact: true }).click();
  };
  const nameField = page.getByLabel("Name*", { exact: true });

  await openEditor("devices", "Catania Weather Station");
  await expect(nameField).toBeDisabled();
  await page.getByRole("link", { name: "Simulation", exact: true }).click();
  await stop.click();
  await expect(start).toBeEnabled();

  await openEditor("devices", "Catania Weather Station");
  await nameField.click();
  await expect(nameField).toBeFocused();

  await page.getByRole("link", { name: "Simulation", exact: true }).click();
  await start.click();
  await expect(stop).toBeVisible();
  await openEditor("gateways", "Catania Physical Gateway");
  await expect(nameField).toBeDisabled();
  await page.getByRole("link", { name: "Simulation", exact: true }).click();
  await stop.click();
  await expect(start).toBeEnabled();

  await openEditor("gateways", "Catania Physical Gateway");
  await nameField.click();
  await expect(nameField).toBeFocused();
});
