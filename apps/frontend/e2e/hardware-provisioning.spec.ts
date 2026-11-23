import { expect, test, type Page } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

const selectFormOption = async (
  page: Page,
  label: string,
  option: string,
) => {
  const field = page
    .locator(".form-field-wrapper")
    .filter({ has: page.locator(".field-label", { hasText: label }) });
  await field.locator(".selected-box").click();
  await field.locator(".choice").filter({ hasText: option }).last().click();
  await expect(field.locator(".selected-box")).toHaveText(option);
};

const selectRadioOption = async (page: Page, id: string) => {
  await page.locator(".choice", { has: page.locator(`#${id}`) }).click();
};

test("creates a virtual gateway from the hardware form", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/gateways/new");

  await page.getByLabel("Name").fill("Catania Lab Gateway");
  await page.getByText("VIRTUAL", { exact: true }).click();
  await page.getByLabel("MAC Address").fill("02:00:00:00:00:01");
  await page.getByLabel("Gateway EUI").fill("0102030405060708");
  await page.getByLabel("Keep Alive (seconds)").fill("30");
  await page.getByLabel("Latitude").fill("37.5079");
  await page.getByLabel("Longitude").fill("15.083");
  await page.getByLabel("Altitude").fill("20");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/gateways\/?$/);
});

test("creates a real gateway with its network endpoint", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/gateways/new");

  await page.getByLabel("Name").fill("Catania Physical Gateway");
  await selectRadioOption(page, "type-real");
  await page.getByLabel("MAC Address").fill("02:00:00:00:00:02");
  await page.getByLabel("Gateway EUI").fill("0102030405060002");
  await page.getByLabel("Gateway IPv4").fill("192.168.1.20");
  await page.getByLabel("Gateway Port").fill("1700");
  await page.getByLabel("Latitude").fill("37.5079");
  await page.getByLabel("Longitude").fill("15.083");
  await page.getByLabel("Altitude").fill("25");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/gateways\/?$/);
});

const fillDeviceLocationAndRadioFields = async (
  page: Page,
  name: string,
  deviceClass: "a" | "b" | "c" = "a",
) => {
  await page.getByLabel("Name").fill(name);
  await page.getByLabel("DevEUI").fill("70B3D57ED0000002");
  await page.getByLabel("Latitude").fill("37.5079");
  await page.getByLabel("Longitude").fill("15.083");
  await page.getByLabel("Altitude").fill("15");
  await selectFormOption(page, "Region", "EU868");
  await selectRadioOption(page, `class-class-${deviceClass}`);
  await page.getByLabel("RX1 Delay").fill("1000");
  await page.getByLabel("RX1 Duration").fill("3000");
  await selectFormOption(page, "RX1 Data Rate offset", "0");
  await page.getByLabel("RX2 Delay").fill("2000");
  await page.getByLabel("RX2 Duration").fill("3000");
  await page.getByLabel("RX2 Channel Frequency").fill("869525000");
  await selectFormOption(page, "RX2 Data Rate", "1");
  await page.getByLabel("RX2 ACK Timeout").fill("2000");
  await page.getByLabel("FPort").fill("1");
  await page.getByLabel("Retransmission").fill("1");
  await page.getByLabel("Uplink Interval").fill("10");
  await selectRadioOption(page, "oversizedPayloadBehavior-truncate");
  await selectRadioOption(page, "MType-unconfirmed-data-up");
  await page.getByRole("textbox", { name: "Payload*" }).fill("Hello LoRaWAN");
  await page.getByLabel("Antenna Range").fill("10000");
};

test("creates an OTAA class A device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania OTAA Sensor");
  await page.locator('input[name="payloadCodecEnabled"]').locator("..").click();
  await expect(page.getByLabel("JavaScript Payload Codec")).toHaveValue(
    "function decodeUplink(input) {\n  return { data: {} };\n}",
  );
  await selectRadioOption(page, "activation-oota");
  await page.getByLabel("JoinEUI").fill("0000000000000001");
  await page.getByLabel("AppKey").fill("00000000000000000000000000000001");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("creates an OTAA class B device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania OTAA Class B Sensor", "b");
  await selectRadioOption(page, "activation-oota");
  await page.getByLabel("JoinEUI").fill("0000000000000001");
  await page.getByLabel("AppKey").fill("00000000000000000000000000000001");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("creates an OTAA class C device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania OTAA Class C Sensor", "c");
  await selectRadioOption(page, "activation-oota");
  await page.getByLabel("JoinEUI").fill("0000000000000001");
  await page.getByLabel("AppKey").fill("00000000000000000000000000000001");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("creates an ABP class A device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania ABP Sensor");
  await selectRadioOption(page, "activation-abp");
  await page.getByLabel("DevAddr").fill("26011BDA");
  await page.getByLabel("NwkSKey").fill("00000000000000000000000000000002");
  await page.getByLabel("AppSKey").fill("00000000000000000000000000000003");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("creates an ABP class B device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania ABP Class B Sensor", "b");
  await selectRadioOption(page, "activation-abp");
  await page.getByLabel("DevAddr").fill("26011BDA");
  await page.getByLabel("NwkSKey").fill("00000000000000000000000000000002");
  await page.getByLabel("AppSKey").fill("00000000000000000000000000000003");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("creates an ABP class C device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await fillDeviceLocationAndRadioFields(page, "Catania ABP Class C Sensor", "c");
  await selectRadioOption(page, "activation-abp");
  await page.getByLabel("DevAddr").fill("26011BDA");
  await page.getByLabel("NwkSKey").fill("00000000000000000000000000000002");
  await page.getByLabel("AppSKey").fill("00000000000000000000000000000003");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

test("loads dependent RX fields when editing a device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");
  await page.goto("/hardware/devices/new?deviceId=device-1");

  const rx1DataRateOffset = page
    .locator(".form-field-wrapper")
    .filter({
      has: page.locator(".field-label", { hasText: "RX1 Data Rate offset" }),
    })
    .locator(".selected-box");

  await expect(page.getByLabel("RX2 Channel Frequency")).toHaveValue(
    "869525000",
  );
  await expect(page.locator(".pos-marker")).toBeVisible();
  await expect(
    page
      .locator(".form-field-wrapper")
      .filter({ has: page.locator(".field-label", { hasText: "RX2 Data Rate" }) })
      .locator(".selected-box"),
  ).toHaveText("0");
  await expect(rx1DataRateOffset).toHaveText("0");
});

test("loads and saves an edited device", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");
  await page.goto("/hardware/devices/new?deviceId=device-1");

  await expect(page.getByLabel("Name")).toHaveValue("Catania Weather Station");
  await expect(page.getByLabel("RX2 Channel Frequency")).toHaveValue(
    "869525000",
  );
  await expect(page.getByLabel("Payload Codec", { exact: true })).toBeChecked();
  await expect(page.getByLabel("JavaScript Payload Codec")).toHaveValue(
    "function decodeUplink(input) { return { data: {} }; }",
  );
  await page.getByLabel("Name").fill("Catania Weather Station Updated");
  await page
    .getByLabel("JavaScript Payload Codec")
    .fill("function decodeUplink(input) { return { data: { ok: true } }; }");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
  await page.goto("/hardware/devices/new?deviceId=device-1");
  await expect(page.getByLabel("JavaScript Payload Codec")).toHaveValue(
    "function decodeUplink(input) { return { data: { ok: true } }; }",
  );
});

test("prevents saving a payload codec with invalid JavaScript", async ({
  page,
}) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");
  await page.goto("/hardware/devices/new?deviceId=device-1");

  await page
    .getByLabel("JavaScript Payload Codec")
    .fill("function decodeUplink(input) { return { data: }; }");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page.getByText("Enter valid JavaScript syntax")).toBeVisible();
  await expect(page).toHaveURL(/deviceId=device-1/);
});

for (const script of [
  '// function decodeUplink(input) {}\nconst unrelated = 1;',
  'const description = "function decodeUplink(input) {}";',
  'async function decodeUplink(input) { return { data: {} }; }',
]) {
  test(`rejects a codec without a synchronous top-level decoder: ${script}`, async ({ page }) => {
    await mockBackend(page);
    await page.goto("/hardware/devices");
    await page.goto("/hardware/devices/new?deviceId=device-1");
    await page.getByLabel("JavaScript Payload Codec").fill(script);
    await expect(page.getByText("Define a decodeUplink(input) function", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Save", exact: true })).toBeDisabled();
    await page.getByLabel("JavaScript Payload Codec").fill("const decodeUplink = input => ({ data: { first: input.bytes[0] } });");
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
    await page.goto("/hardware/devices/new?deviceId=device-1");
    await expect(page.getByLabel("JavaScript Payload Codec")).toHaveValue("const decodeUplink = input => ({ data: { first: input.bytes[0] } });");
  });
}

test("creates a Raw LoRa device and preserves its receive mode", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");
  await fillDeviceLocationAndRadioFields(page, "Raw LoRa demonstration");
  await selectRadioOption(page, "activation-abp");
  await page.getByLabel("DevAddr").fill("26011BDA");
  await page.getByLabel("NwkSKey").fill("00000000000000000000000000000002");
  await page.getByLabel("AppSKey").fill("00000000000000000000000000000003");
  await selectFormOption(page, "Radio Protocol", "Raw LoRa");
  await selectFormOption(page, "Radio Receive Mode", "Auto");
  const submitted = page.waitForRequest(request => request.url().endsWith("/device/create-device") && request.method() === "POST");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  expect((await submitted).postDataJSON()).toMatchObject({ radioProtocol: "raw-lora", radioReceiveMode: "auto" });
  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
});

for (const deviceCodec of ["empty", "absent"] as const) {
  test(`loads the default codec when the saved codec is ${deviceCodec}`, async ({
    page,
  }) => {
    await mockBackend(page, { deviceCodec });
    await page.goto("/hardware/devices");
    await page.goto("/hardware/devices/new?deviceId=device-1");

    if (deviceCodec === "absent") {
      await page
        .locator('input[name="payloadCodecEnabled"]')
        .locator("..")
        .click();
    }

    await expect(page.getByLabel("JavaScript Payload Codec")).toHaveValue(
      "function decodeUplink(input) {\n  return { data: {} };\n}",
    );
  });
}

test("updates device status, position and payload during simulation", async ({ page }) => {
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  await page.getByRole("button", { name: "Start simulation" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();

  await page.goto("/hardware/devices");
  await page.goto("/hardware/devices/new?deviceId=device-1");
  const activeField = page.locator('input[name="active"]');
  const activeControl = page
    .locator(".boolean-checkbox-form-field")
    .filter({ has: activeField });
  await expect(activeField).toBeChecked();
  await expect(page.getByLabel("Name")).toBeDisabled();

  await expect(page.getByRole("textbox", { name: "Payload*" })).toBeEnabled();
  await expect(page.locator('input[name="base64Encoded"]')).toBeEnabled();
  await page.getByRole("textbox", { name: "Payload*" }).fill("TmV3IHBheWxvYWQ=");
  await expect(activeField).toBeEnabled();
  await page.getByLabel("Latitude").fill("37.5079");
  await page.getByLabel("Longitude").fill("15.0830");
  await page.getByLabel("Altitude").fill("125");
  await activeControl.locator(".box").click();
  await expect(activeField).not.toBeChecked();
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
  await expect(page.getByText("inactive", { exact: true })).toBeVisible();

  await page.goto("/hardware/devices/new?deviceId=device-1");
  await expect(activeField).not.toBeChecked();
  await expect(page.getByLabel("Latitude")).toHaveValue("37.5079");
  await expect(page.getByLabel("Longitude")).toHaveValue("15.083");
  await expect(page.getByLabel("Altitude")).toHaveValue("125");
  await expect(page.getByRole("textbox", { name: "Payload*" })).toHaveValue("TmV3IHBheWxvYWQ=");
  const base64Field = page.locator('input[name="base64Encoded"]');
  await expect(base64Field).toBeChecked();
  await page.locator(".boolean-checkbox-form-field").filter({ has: base64Field }).locator(".box").click();
  await expect(base64Field).not.toBeChecked();
  await page.getByRole("textbox", { name: "Payload*" }).fill("Live reading");
  await activeControl.locator(".box").click();
  await expect(activeField).toBeChecked();
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/\/hardware\/devices\/?$/);
  await expect(page.getByText("active", { exact: true })).toBeVisible();
  await page.goto("/hardware/devices/new?deviceId=device-1");
  await expect(page.getByRole("textbox", { name: "Payload*" })).toHaveValue("Live reading");
  await expect(base64Field).not.toBeChecked();
  await page.goto("/simulation/dashboard");
  await expect(page.getByText("Running", { exact: true })).toBeVisible();
});

test("updates gateway status and position during simulation", async ({
  page,
}) => {
  await mockBackend(page, { devices: "active" });
  await page.goto("/simulation/dashboard");

  await page.getByRole("button", { name: "Start simulation" }).click();
  await expect(page.getByText("Running", { exact: true })).toBeVisible();

  await page.goto("/hardware/gateways");
  await page.goto("/hardware/gateways/new?gatewayId=gateway-1");
  const activeField = page.locator('input[name="active"]');
  const activeControl = page
    .locator(".boolean-checkbox-form-field")
    .filter({ has: activeField });

  await expect(activeField).toBeChecked();
  await expect(page.getByLabel("Name")).toBeDisabled();
  await expect(page.getByLabel("Gateway IPv4")).toBeDisabled();
  await expect(activeField).toBeEnabled();
  await page.getByLabel("Latitude").fill("37.5079");
  await page.getByLabel("Longitude").fill("15.0830");
  await page.getByLabel("Altitude").fill("125");
  await activeControl.locator(".box").click();
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/\/hardware\/gateways\/?$/);
  await expect(page.getByText("inactive", { exact: true })).toBeVisible();
  await page.goto("/hardware/gateways/new?gatewayId=gateway-1");
  await expect(page.getByLabel("Latitude")).toHaveValue("37.5079");
  await expect(page.getByLabel("Longitude")).toHaveValue("15.083");
  await expect(page.getByLabel("Altitude")).toHaveValue("125");
});

test("loads and saves an edited real gateway", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/gateways");
  await page.goto("/hardware/gateways/new?gatewayId=gateway-1");

  await expect(page.getByLabel("Name")).toHaveValue("Catania Physical Gateway");
  await expect(page.getByLabel("Gateway IPv4")).toHaveValue("192.168.1.20");
  await expect(page.getByLabel("Gateway Port")).toHaveValue("1700");
  await page.getByLabel("Name").fill("Catania Physical Gateway Updated");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/\/hardware\/gateways\/?$/);
});

test("keeps invalid device identifiers from being submitted", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/hardware/devices/new");

  await page.getByLabel("DevEUI").fill("1234");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page.getByText("Invalid DevEUI", { exact: true })).toBeVisible();
  await expect(page).toHaveURL(/\/hardware\/devices\/new\/?$/);
});
