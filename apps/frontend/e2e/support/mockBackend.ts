import type { Page } from "@playwright/test";

export type MockBackendOptions = {
  logs?: "success" | "empty" | "error";
  devices?: "empty" | "active" | "raw";
  deviceCodec?: "configured" | "empty" | "absent";
  bridgeEnabled?: boolean;
};

const profile = { id: "profile-1", name: "Default profile" };
const secondProfile = { id: "profile-2", name: "Test profile" };

const activeDevice = {
  id: "device-1",
  active: true,
  name: "Catania Weather Station",
  devEUI: "70B3D57ED0000001",
  class: "class-a",
  activation: "oota",
  OOTAConfig: {
    joinEUI: "0000000000000001",
    appKey: "00000000000000000000000000000001",
  },
  locationConfig: {
    latitude: 37.5079,
    longitude: 15.083,
    altitude: 15,
    region: "EU868",
  },
  RX1Config: { delay: 1, duration: 1, dataRateOffset: 0 },
  RX2Config: {
    delay: 2,
    duration: 1,
    channelFrequency: 869525000,
    dataRate: 0,
    ACKTimeout: 2000,
  },
  advancedConfig: { antennaRange: 5000, ADREnabled: true },
  frameConfig: {
    fPort: 1,
    retransmission: 1,
    disableFrameCounterValidation: false,
  },
  payloadConfig: {
    uplinkInterval: 60000,
    oversizedPayloadBehavior: "truncate",
    MType: "unconfirmed-data-up",
    payload: "AQID",
    base64Encoded: true,
    payloadCodec: {
      enabled: true,
      script: "function decodeUplink(input) { return { data: {} }; }",
    },
  },
};

const activeGateway = {
  id: "gateway-1",
  active: true,
  name: "Catania Lab Gateway",
  type: "virtual",
  gatewayEUI: "0102030405060708",
  keepAlive: 30,
  latitude: 37.5079,
  longitude: 15.083,
  altitude: 20,
};

const editableGateway = {
  ...activeGateway,
  name: "Catania Physical Gateway",
  type: "real",
  gatewayIPv4: "192.168.1.20",
  gatewayPort: 1700,
  altitude: 25,
};

const runs = [
  {
    summary: {
      id: "run-new",
      startedAtMilliseconds: 1_800_000_000_000,
      endedAtMilliseconds: 1_800_000_001_000,
      durationMilliseconds: 1_000,
      status: "stopped",
      speed: 1,
      seed: 42,
      deviceCount: 2,
      gatewayCount: 1,
      eventCount: 12,
      packetSuccessRate: 0.75,
    },
    objects: [],
  },
  {
    summary: {
      id: "run-old",
      startedAtMilliseconds: 1_700_000_000_000,
      endedAtMilliseconds: 1_700_000_001_000,
      durationMilliseconds: 1_000,
      status: "failed",
      speed: 1,
      seed: 7,
      deviceCount: 1,
      gatewayCount: 1,
      eventCount: 3,
      packetSuccessRate: 0,
    },
    objects: [],
  },
];

const json = (data: unknown) => ({
  status: 200,
  contentType: "application/json",
  headers: { "access-control-allow-origin": "*" },
  body: JSON.stringify(data),
});

export const mockBackend = async (
  page: Page,
  options: MockBackendOptions = {},
): Promise<void> => {
  const hasDevices = options.devices === "active" || options.devices === "raw";
  let gatewayBridge = {
    enabled: options.bridgeEnabled ?? false,
    address: "0.0.0.0",
    port: 1700,
  };
  let profiles = [profile, secondProfile];
  let simulationStatus = "idle";
  const runtimeDevicePayloadConfig: Omit<
    typeof activeDevice.payloadConfig,
    "payloadCodec"
  > & {
    payloadCodec?: typeof activeDevice.payloadConfig.payloadCodec;
  } = { ...activeDevice.payloadConfig };

  if (options.deviceCodec === "empty") {
    runtimeDevicePayloadConfig.payloadCodec = { enabled: true, script: "" };
  } else if (options.deviceCodec === "absent") {
    delete runtimeDevicePayloadConfig.payloadCodec;
  }

  let runtimeDevice = {
    ...activeDevice,
    ...(options.devices === "raw"
      ? { radioProtocol: "raw-lora", radioReceiveMode: "raw-lora" }
      : { radioProtocol: "lorawan", radioReceiveMode: "lorawan" }),
    payloadConfig: runtimeDevicePayloadConfig,
  };
  let runtimeGateway = { ...editableGateway };

  const simulationSnapshot = () => ({
    state: {
      status: simulationStatus,
      speed: 1,
      elapsedMilliseconds: simulationStatus === "idle" ? 0 : 1000,
      eventSequence: simulationStatus === "idle" ? 0 : 1,
      deviceCount: hasDevices ? 1 : 0,
      gatewayCount: hasDevices ? 1 : 0,
    },
    devices: [],
    gateways: [],
    metrics: {
      totalUplinks: 0,
      successfulUplinks: 0,
      droppedUplinks: 0,
      totalPacketsReceived: 0,
      packetSuccessRate: 0,
      activeDevices: hasDevices ? 1 : 0,
      activeGateways: hasDevices ? 1 : 0,
    },
  });

  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    const isBackendRequest = url.pathname.startsWith("/api/");
    if (!isBackendRequest) {
      await route.continue();
      return;
    }

    const path = url.pathname.replace(/^\/api/, "");

    if (path.endsWith("/profile/get-profiles")) {
      await route.fulfill(json({ profiles, total: profiles.length }));
      return;
    }

    if (path.endsWith("/profile/create-profile")) {
      const body = route.request().postDataJSON() as { name?: string };
      const createdProfile = {
        id: `profile-${profiles.length + 1}`,
        name: body.name ?? "New profile",
      };
      profiles = [...profiles, createdProfile];
      await route.fulfill(json({ profile: createdProfile }));
      return;
    }

    if (path.includes("/profile/update-profile/")) {
      const body = route.request().postDataJSON() as { name?: string };
      const id = decodeURIComponent(path.split("/").at(-1) ?? "");
      const updatedProfile = profiles.find((item) => item.id === id);
      if (!updatedProfile) {
        await route.fulfill(json({ error: "Profile not found" }));
        return;
      }
      const nextProfile = { ...updatedProfile, name: body.name ?? updatedProfile.name };
      profiles = profiles.map((item) => (item.id === id ? nextProfile : item));
      await route.fulfill(json({ profile: nextProfile }));
      return;
    }

    if (path.includes("/profile/delete-profile/")) {
      const id = decodeURIComponent(path.split("/").at(-1) ?? "");
      profiles = profiles.filter((item) => item.id !== id);
      await route.fulfill(json({}));
      return;
    }

    if (path.includes("/profile/export-profile/")) {
      const id = decodeURIComponent(path.split("/").at(-1) ?? "");
      const exportedProfile = profiles.find((item) => item.id === id) ?? profile;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin": "*",
          "content-disposition": "attachment; filename=profile.json",
        },
        body: JSON.stringify({
          format: "lwn-profile",
          version: 1,
          profile: exportedProfile,
          devices: [],
          gateways: [],
          logs: [],
        }),
      });
      return;
    }

    if (path.endsWith("/profile/import-profile")) {
      const body = route.request().postDataJSON() as {
        archive?: { profile?: { name?: string } };
      };
      const importedProfile = {
        id: `profile-imported-${profiles.length + 1}`,
        name: body.archive?.profile?.name ?? "Imported profile",
      };
      profiles = [...profiles, importedProfile];
      await route.fulfill(json({ profile: importedProfile }));
      return;
    }

    if (path.endsWith("/simulation/activity")) {
      await route.fulfill(
        json({
          active: simulationStatus === "running" || simulationStatus === "paused",
          status: simulationStatus,
        }),
      );
      return;
    }

    if (path.endsWith("/device/get-devices")) {
      await route.fulfill(
        json({ devices: hasDevices ? [runtimeDevice] : [] }),
      );
      return;
    }

    if (path.endsWith("/device/get-device/device-1")) {
      await route.fulfill(json({ device: runtimeDevice }));
      return;
    }

    if (path.includes("/device/update-device/")) {
      const body = route.request().postDataJSON() as {
        device?: Record<string, unknown>;
      };
      runtimeDevice = {
        ...runtimeDevice,
        ...(body.device ?? {}),
        id: "device-1",
      };
      await route.fulfill(json({ device: runtimeDevice }));
      return;
    }

    if (path.endsWith("/gateway/get-gateways")) {
      await route.fulfill(
        json({ gateways: hasDevices ? [runtimeGateway] : [] }),
      );
      return;
    }

    if (path.endsWith("/gateway/get-gateway/gateway-1")) {
      await route.fulfill(json({ gateway: runtimeGateway }));
      return;
    }

    if (path.includes("/gateway/update-gateway/")) {
      const body = route.request().postDataJSON() as {
        gateway?: Record<string, unknown>;
      };
      runtimeGateway = {
        ...runtimeGateway,
        ...(body.gateway ?? {}),
        id: "gateway-1",
      };
      await route.fulfill(
        json({ gateway: runtimeGateway }),
      );
      return;
    }

    if (path.endsWith("/gateway/create-gateway")) {
      await route.fulfill(json({ gateway: { ...activeGateway, id: "gateway-new" } }));
      return;
    }

    if (path.endsWith("/device/create-device")) {
      await route.fulfill(json({ device: { ...activeDevice, id: "device-new" } }));
      return;
    }

    if (path.endsWith("/configuration/gateway-bridge")) {
      if (route.request().method() !== "GET") {
        try {
          gatewayBridge = {
            ...gatewayBridge,
            ...(route.request().postDataJSON() as Partial<typeof gatewayBridge>),
          };
        } catch {
          // Keep the previous bridge state when the request has no JSON body.
        }
      }

      await route.fulfill(json(gatewayBridge));
      return;
    }

    if (path.endsWith("/simulation/logs/telemetry")) {
      await route.fulfill({ status: 200, contentType: "text/csv", body: "run_id,packet_success_rate_pct\nrun-new,80\n" });
      return;
    }

    if (/\/simulation\/logs\/[^/]+\/telemetry$/.test(path)) {
      await route.fulfill({ status: 200, contentType: "application/zip", body: Buffer.from("PK-telemetry-archive") });
      return;
    }

    if (path.endsWith("/simulation/logs/export")) {
      await route.fulfill({ status: 200, contentType: "application/zip", body: Buffer.from("PK-test-archive") });
      return;
    }

    if (/\/simulation\/logs\/[^/]+\/export$/.test(path)) {
      await route.fulfill(json({ run: runs[0], events: [{ id: "event-1", sequence: 1, type: "device-uplink-transmitted", message: "Exported uplink", timestampMilliseconds: 0 }] }));
      return;
    }

    if (path.endsWith("/simulation/logs")) {
      if (options.logs === "error") {
        await route.fulfill({
          status: 503,
          contentType: "application/json",
          headers: { "access-control-allow-origin": "*" },
          body: JSON.stringify({ error: "Backend unavailable" }),
        });
        return;
      }

      if (options.logs === "empty") {
        await route.fulfill(json({ runs: [], total: 0 }));
        return;
      }

      await route.fulfill(json({ runs, total: runs.length }));
      return;
    }

    if (/\/simulation\/logs\/[^/]+\/events$/.test(path)) {
      await route.fulfill(json({ events: [], lastSequence: 0 }));
      return;
    }

    if (/\/simulation\/logs\/[^/]+$/.test(path)) {
      await route.fulfill(json(runs.find((run) => path.endsWith(`/${run.summary.id}`)) ?? runs[0]));
      return;
    }

    if (path.endsWith("/simulation/snapshot")) {
      await route.fulfill(json(simulationSnapshot()));
      return;
    }

    if (path.endsWith("/simulation/start")) {
      simulationStatus = "running";
      await route.fulfill(json(simulationSnapshot()));
      return;
    }

    if (path.endsWith("/simulation/pause")) {
      simulationStatus = "paused";
      await route.fulfill(json(simulationSnapshot()));
      return;
    }

    if (path.endsWith("/simulation/resume")) {
      simulationStatus = "running";
      await route.fulfill(json(simulationSnapshot()));
      return;
    }

    if (path.endsWith("/simulation/stop")) {
      simulationStatus = "stopped";
      await route.fulfill(json(simulationSnapshot()));
      return;
    }

    if (path.endsWith("/simulation/uplinks")) {
      await route.fulfill(json({ id: "action-uplink" }));
      return;
    }

    if (path.endsWith("/simulation/downlinks")) {
      await route.fulfill(json({ id: "action-downlink" }));
      return;
    }

    if (path.endsWith("/simulation/mac-commands")) {
      await route.fulfill(json({ id: "action-mac-command" }));
      return;
    }

    if (path.endsWith("/simulation/events")) {
      await route.fulfill(json({ events: [], lastSequence: 0 }));
      return;
    }

    await route.fulfill(json({}));
  });
};
