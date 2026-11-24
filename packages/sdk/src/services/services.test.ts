import type {
  AppInfoResponse,
  Gateway,
  GatewayBridgeConfig,
  Profile,
  ProfileArchive,
  SimulationActivity,
  StatusResponse,
} from "@lwn-simulator/contracts";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiCaller } from "../models/ApiCaller";
import { AppInfoService } from "./AppInfoService/AppInfoService";
import { GatewayService } from "./GatewayService/GatewayService";
import { ProfileConfigurationService } from "./ProfileConfigurationService/ProfileConfigurationService";
import { ProfileService } from "./ProfileService/ProfileService";

const gateway = { id: "gateway-1", name: "Catania Gateway" } as Gateway;
const profile = { id: "profile-1", name: "Demo profile" } as Profile;
const archive = { profile } as unknown as ProfileArchive;
const bridge = { enabled: true, address: "127.0.0.1", port: 1700 } as GatewayBridgeConfig;

afterEach(() => vi.restoreAllMocks());

describe("SDK service API routing", () => {
  it("routes gateway CRUD calls to the expected endpoints", async () => {
    const getSpy = vi.spyOn(ApiCaller.prototype, "get");
    getSpy
      .mockResolvedValueOnce({ gateway } as never)
      .mockResolvedValueOnce({ gateways: [gateway] } as never);
    const postSpy = vi
      .spyOn(ApiCaller.prototype, "post")
      .mockResolvedValue({ gateway } as never);
    const putSpy = vi
      .spyOn(ApiCaller.prototype, "put")
      .mockResolvedValue({ gateway } as never);
    const deleteSpy = vi
      .spyOn(ApiCaller.prototype, "delete")
      .mockResolvedValue(undefined as never);
    const service = new GatewayService("http://localhost:8000");

    await service.getGateway({ id: "gateway/1" });
    await service.getGateways();
    await service.createGateway({} as never);
    await service.updateGateway({ id: gateway.id, gateway });
    await service.deleteGateway({ id: gateway.id });

    expect(getSpy).toHaveBeenNthCalledWith(1, "/get-gateway/gateway%2F1");
    expect(getSpy).toHaveBeenNthCalledWith(2, "/get-gateways");
    expect(postSpy).toHaveBeenCalledWith("/create-gateway", {});
    expect(putSpy).toHaveBeenCalledWith(`/update-gateway/${gateway.id}`, {
      id: gateway.id,
      gateway,
    });
    expect(deleteSpy).toHaveBeenCalledWith(`/delete-gateway/${gateway.id}`);
  });

  it("routes profile import/export and simulation activity calls", async () => {
    const profileGetSpy = vi.spyOn(ApiCaller.prototype, "get");
    profileGetSpy
      .mockResolvedValueOnce({ profiles: [profile] } as never)
      .mockResolvedValueOnce({ profile } as never)
      .mockResolvedValueOnce(archive as never)
      .mockResolvedValueOnce({ active: false, status: "idle" } as SimulationActivity);
    const postSpy = vi
      .spyOn(ApiCaller.prototype, "post")
      .mockResolvedValueOnce({ profile } as never)
      .mockResolvedValueOnce({ profile } as never);
    const putSpy = vi
      .spyOn(ApiCaller.prototype, "put")
      .mockResolvedValue({ profile } as never);
    const deleteSpy = vi
      .spyOn(ApiCaller.prototype, "delete")
      .mockResolvedValue(undefined as never);
    const service = new ProfileService("http://localhost:8000");

    await service.getProfiles();
    await service.getProfile(profile.id);
    await service.createProfile({ name: profile.name });
    await service.updateProfile({ id: profile.id, name: profile.name });
    await service.deleteProfile(profile.id);
    await service.exportProfile(profile.id);
    await service.importProfile(archive);
    await service.getSimulationActivity();

    expect(profileGetSpy).toHaveBeenNthCalledWith(1, "/get-profiles");
    expect(profileGetSpy).toHaveBeenNthCalledWith(
      2,
      `/get-profile/${profile.id}`,
    );
    expect(profileGetSpy).toHaveBeenNthCalledWith(
      3,
      `/export-profile/${profile.id}`,
      { timeout: 120000 },
    );
    expect(profileGetSpy).toHaveBeenNthCalledWith(4, "/activity");
    expect(postSpy).toHaveBeenNthCalledWith(1, "/create-profile", {
      name: profile.name,
    });
    expect(postSpy).toHaveBeenNthCalledWith(2, "/import-profile", { archive });
    expect(putSpy).toHaveBeenCalledWith(`/update-profile/${profile.id}`, {
      id: profile.id,
      name: profile.name,
    });
    expect(deleteSpy).toHaveBeenCalledWith(`/delete-profile/${profile.id}`);
  });

  it("routes application and bridge configuration calls", async () => {
    const getSpy = vi.spyOn(ApiCaller.prototype, "get");
    getSpy
      .mockResolvedValueOnce({ port: "8000" } as StatusResponse)
      .mockResolvedValueOnce({ app: "LWN Simulator", version: "1.0.0" } as AppInfoResponse)
      .mockResolvedValueOnce(bridge as never);
    const putSpy = vi
      .spyOn(ApiCaller.prototype, "put")
      .mockResolvedValue(bridge as never);
    const appInfo = new AppInfoService("http://localhost:8000");
    const configuration = new ProfileConfigurationService(
      "http://localhost:8000",
    );

    await appInfo.status();
    await appInfo.appInfo();
    await configuration.getGatewayBridge();
    await configuration.updateGatewayBridge(bridge);

    expect(getSpy).toHaveBeenNthCalledWith(1, "/status");
    expect(getSpy).toHaveBeenNthCalledWith(2, "/info");
    expect(getSpy).toHaveBeenNthCalledWith(3, "/gateway-bridge");
    expect(putSpy).toHaveBeenCalledWith("/gateway-bridge", bridge);
  });
});
