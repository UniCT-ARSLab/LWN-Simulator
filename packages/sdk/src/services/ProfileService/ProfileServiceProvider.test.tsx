import { act, renderHook } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, expect, it, vi } from "vitest";
import type { SimulationActivity } from "@lwn-simulator/contracts";
import { ApiCaller } from "../../models/ApiCaller";
import { installWebSocketMock, TestWebSocket } from "../../test/webSocket";
import WebSocketServiceProvider from "../WebSocketService/WebSocketServiceProvider";
import ProfileServiceProvider from "./ProfileServiceProvider";
import { useProfileService } from "./useProfileService";

const baseUrl = "http://localhost:8000";
const wrapper = ({ children }: PropsWithChildren) => (
  <WebSocketServiceProvider baseUrl={baseUrl}>
    <ProfileServiceProvider baseUrl={baseUrl}>{children}</ProfileServiceProvider>
  </WebSocketServiceProvider>
);

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("refreshes activity on reconnect after missing a stop notification", async () => {
  vi.useFakeTimers();
  installWebSocketMock();
  let activity: SimulationActivity = { active: true, status: "running", profileID: "profile-1" };
  vi.spyOn(ApiCaller.prototype, "get").mockImplementation(async (path) =>
    (path === "/activity" ? activity : { profiles: [] }) as never,
  );
  const { result } = renderHook(() => useProfileService(), { wrapper });

  await act(async () => {
    TestWebSocket.latest().open();
  });
  expect(result.current.simulationActivity).toMatchObject({ active: true, profileID: "profile-1" });

  act(() => TestWebSocket.latest().serverClose());
  activity = { active: false, status: "idle" };

  act(() => vi.advanceTimersByTime(1000));
  await act(async () => {
    TestWebSocket.latest().open();
  });

  expect(result.current.simulationActivity).toEqual({ active: false, status: "idle" });
});

it("selects a surviving profile after reconnecting with browser storage blocked", async () => {
  vi.useFakeTimers();
  installWebSocketMock();
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
    throw new DOMException("Storage is disabled", "SecurityError");
  });
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
    throw new DOMException("Storage is disabled", "SecurityError");
  });
  let profiles = [{ id: "first", name: "Original" }];
  vi.spyOn(ApiCaller.prototype, "get").mockImplementation(async (path) =>
    (path === "/activity" ? { active: false, status: "idle" } : { profiles }) as never,
  );
  const { result } = renderHook(() => useProfileService(), { wrapper });
  await act(async () => TestWebSocket.latest().open());
  expect(result.current.activeProfile?.id).toBe("first");

  act(() => TestWebSocket.latest().serverClose());
  await act(async () => { vi.advanceTimersByTime(1000); });
  profiles = [{ id: "second", name: "Replacement" }];
  await act(async () => TestWebSocket.latest().open());
  expect(result.current.profiles).toEqual(profiles);
  expect(result.current.activeProfileID).toBe("second");
  expect(result.current.error).toBeNull();
});
