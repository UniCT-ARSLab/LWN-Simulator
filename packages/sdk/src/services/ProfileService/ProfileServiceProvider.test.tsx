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
