import { renderHook, waitFor, act } from "@testing-library/react";
import type { Device, RealtimeWebSocketMessage } from "@lwn-simulator/contracts";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiCaller } from "../../models/ApiCaller";
import WebSocketServiceProvider from "../WebSocketService/WebSocketServiceProvider";
import { useDeviceService } from "./useDeviceService";
import DeviceServiceProvider from "./DeviceServiceProvider";
import { installWebSocketMock, TestWebSocket } from "../../test/webSocket";

const device = {
  id: "device-1",
  name: "Weather Station",
} as Device;

const updatedDevice = { ...device, name: "Updated Weather Station" } as Device;

const wrapper = ({ children }: PropsWithChildren) => (
  <WebSocketServiceProvider baseUrl="http://localhost:8000">
    <DeviceServiceProvider baseUrl="http://localhost:8000" profileID="profile-1">
      {children}
    </DeviceServiceProvider>
  </WebSocketServiceProvider>
);

describe("DeviceServiceProvider", () => {
  beforeEach(() => installWebSocketMock());

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("loads devices and applies realtime updates only for the active profile", async () => {
    vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      devices: [device],
    } as never);
    const { result } = renderHook(() => useDeviceService(), { wrapper });
    TestWebSocket.latest().open();

    await waitFor(() => expect(result.current.devices).toEqual([device]));

    const ignoredMessage: RealtimeWebSocketMessage = {
      type: "device.updated" as RealtimeWebSocketMessage["type"],
      timestampMilliseconds: 1,
      profileID: "other-profile",
      device: updatedDevice,
    };
    act(() => TestWebSocket.latest().message(JSON.stringify(ignoredMessage)));
    expect(result.current.devices).toEqual([device]);

    const acceptedMessage: RealtimeWebSocketMessage = {
      type: "device.updated" as RealtimeWebSocketMessage["type"],
      timestampMilliseconds: 2,
      profileID: "profile-1",
      device: updatedDevice,
    };
    act(() => TestWebSocket.latest().message(JSON.stringify(acceptedMessage)));

    await waitFor(() => expect(result.current.devices).toEqual([updatedDevice]));
  });

  it("updates the local collection after create and delete operations", async () => {
    const getSpy = vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      devices: [],
    } as never);
    const postSpy = vi.spyOn(ApiCaller.prototype, "post").mockResolvedValue({
      device,
    } as never);
    const deleteSpy = vi.spyOn(ApiCaller.prototype, "delete").mockResolvedValue(
      undefined as never,
    );
    const { result } = renderHook(() => useDeviceService(), { wrapper });
    TestWebSocket.latest().open();
    await waitFor(() => expect(getSpy).toHaveBeenCalled());

    await act(async () => {
      await result.current.createDevice({} as never);
    });
    expect(result.current.devices).toEqual([device]);
    expect(postSpy).toHaveBeenCalledWith("/create-device", {});

    act(() =>
      TestWebSocket.latest().message(
        JSON.stringify({
          type: "device.created",
          timestampMilliseconds: 2,
          profileID: "profile-1",
          device,
        }),
      ),
    );
    await waitFor(() => expect(result.current.devices).toEqual([device]));

    await act(async () => {
      await result.current.deleteDevice({ id: device.id });
    });
    expect(result.current.devices).toEqual([]);
    expect(deleteSpy).toHaveBeenCalledWith(`/delete-device/${device.id}`);
  });
});
