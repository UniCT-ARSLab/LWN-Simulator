import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("queues uplink, downlink and MAC commands while the engine is running", async ({
  page,
}) => {
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  await page.getByRole("button", { name: "Start simulation" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Queue Uplink" }).click();
  await expect(page.getByText("Queued · action-uplink", { exact: true })).toBeVisible();

  await page.getByRole("tab", { name: "Downlink" }).click();
  await page.getByLabel("Payload (UTF-8)").fill("hello device");
  await page.getByRole("button", { name: "Queue Downlink" }).click();
  await expect(page.getByText("Queued · action-downlink", { exact: true })).toBeVisible();

  await page.getByRole("tab", { name: "Mac Command" }).click();
  await page.getByRole("button", { name: "Queue Mac Command" }).click();
  await expect(
    page.getByText("Queued · action-mac-command", { exact: true }),
  ).toBeVisible();
});

test("shows downlink validation feedback before sending an invalid payload", async ({
  page,
}) => {
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  await page.getByRole("button", { name: "Start simulation" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();
  await page.getByRole("tab", { name: "Downlink" }).click();
  await page.getByLabel("FPort").fill("224");
  await page.getByRole("button", { name: "Queue Downlink" }).click();

  await expect(
    page.getByText("FPort must be an integer between 1 and 223", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("Queued · action-downlink", { exact: true })).toHaveCount(0);
});

test("queues raw LoRa traffic without exposing LoRaWAN-only controls", async ({
  page,
}) => {
  await mockBackend(page, { devices: "raw" });
  await page.goto("/simulation/dashboard");

  await page.getByRole("button", { name: "Start simulation" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();
  await expect(
    page.getByText("Raw LoRa devices transmit directly without a LoRaWAN join."),
  ).toBeVisible();

  await page.getByRole("button", { name: "Queue Uplink" }).click();
  await expect(page.getByText("Queued · action-uplink", { exact: true })).toBeVisible();

  await page.getByRole("tab", { name: "Downlink" }).click();
  await page.getByLabel("Payload (UTF-8)").fill("raw payload");
  await expect(page.getByLabel("FPort", { exact: true })).toHaveCount(0);
  await expect(page.getByLabel("Confirmed", { exact: true })).toHaveCount(0);
  await expect(page.getByLabel("ACK", { exact: true })).toHaveCount(0);
  const downlinkRequest = page.waitForRequest(
    (request) => request.url().endsWith("/simulation/downlinks") && request.method() === "POST",
  );
  await page.getByRole("button", { name: "Queue Downlink" }).click();
  expect((await downlinkRequest).postDataJSON()).toEqual({
    deviceID: "device-1",
    payload: Buffer.from("raw payload").toString("base64"),
  });
  await expect(page.getByText("Queued · action-downlink", { exact: true })).toBeVisible();

  await page.getByRole("tab", { name: "Mac Command" }).click();
  await expect(page.getByText("LoRaWAN device required", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Queue Mac Command" })).toBeDisabled();
});
