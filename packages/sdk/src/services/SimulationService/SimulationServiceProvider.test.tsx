import { renderHook, act } from "@testing-library/react";
import type {
  SimulationEvent,
  SimulationSnapshot,
} from "@lwn-simulator/contracts";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiCaller } from "../../models/ApiCaller";
import WebSocketServiceProvider from "../WebSocketService/WebSocketServiceProvider";
import { useSimulationService } from "./useSimulationService";
import SimulationServiceProvider from "./SimulationServiceProvider";
import { installWebSocketMock, TestWebSocket } from "../../test/webSocket";

const snapshot = {
  state: { status: "running", speed: 1, elapsedMilliseconds: 0 },
  devices: [],
  gateways: [],
  metrics: { totalUplinks: 0, successfulUplinks: 0 },
} as unknown as SimulationSnapshot;

const event = {
  id: "event-1",
  sequence: 1,
  message: "Device uplink transmitted",
} as SimulationEvent;

const wrapper = ({ children }: PropsWithChildren) => (
  <WebSocketServiceProvider baseUrl="http://localhost:8000">
    <SimulationServiceProvider baseUrl="http://localhost:8000" profileID="profile-1">
      {children}
    </SimulationServiceProvider>
  </WebSocketServiceProvider>
);

describe("SimulationServiceProvider", () => {
  beforeEach(() => installWebSocketMock());

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("starts a simulation and exposes its snapshot", async () => {
    vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      events: [],
      lastSequence: 0,
    } as never);
    const postSpy = vi.spyOn(ApiCaller.prototype, "post").mockResolvedValue(
      snapshot as never,
    );
    const { result } = renderHook(() => useSimulationService(), { wrapper });
    TestWebSocket.latest().open();

    await act(async () => {
      await result.current.start({ speed: 1 });
    });

    expect(result.current.snapshot?.state.status).toBe("running");
    expect(result.current.snapshot?.state.speed).toBe(1);
    expect(postSpy).toHaveBeenCalledWith("/start", { speed: 1 });
    expect(result.current.events).toEqual([]);
  });

  it("batches realtime events and ignores events from other profiles", async () => {
    vi.useFakeTimers();
    vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      events: [],
      lastSequence: 0,
    } as never);
    const { result } = renderHook(() => useSimulationService(), { wrapper });
    TestWebSocket.latest().open();

    act(() => {
      TestWebSocket.latest().message(
        JSON.stringify({
          type: "simulation.event",
          profileID: "other-profile",
          timestampMilliseconds: 1,
          event,
        }),
      );
    });
    vi.advanceTimersByTime(100);
    expect(result.current.events).toEqual([]);

    act(() => {
      TestWebSocket.latest().message(
        JSON.stringify({
          type: "simulation.event",
          profileID: "profile-1",
          timestampMilliseconds: 2,
          event,
        }),
      );
      vi.advanceTimersByTime(100);
    });

    expect(result.current.events).toEqual([event]);
  });

  it("keeps the HTTP stop status when flushing an earlier realtime snapshot", async () => {
    vi.useFakeTimers();
    vi.spyOn(ApiCaller.prototype, "post").mockResolvedValue({
      ...snapshot,
      state: { ...snapshot.state, status: "stopped" },
    } as never);
    const { result } = renderHook(() => useSimulationService(), { wrapper });

    act(() => {
      TestWebSocket.latest().open();
      TestWebSocket.latest().message(JSON.stringify({
        type: "simulation.snapshot",
        profileID: "profile-1",
        snapshot,
      }));
      TestWebSocket.latest().message(JSON.stringify({
        type: "simulation.event",
        profileID: "profile-1",
        event,
      }));
    });

    await act(async () => {
      await result.current.stop();
    });

    expect(result.current.snapshot?.state.status).toBe("stopped");
    expect(result.current.events).toEqual([event]);
  });

  it("keeps the displayed simulation clock moving between realtime updates", async () => {
    vi.useFakeTimers();
    vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      events: [],
      lastSequence: 0,
    } as never);
    vi.spyOn(ApiCaller.prototype, "post").mockResolvedValue({
      ...snapshot,
      state: { ...snapshot.state, elapsedMilliseconds: 1000 },
    } as never);
    const { result } = renderHook(() => useSimulationService(), { wrapper });
    TestWebSocket.latest().open();

    await act(async () => {
      await result.current.start({ speed: 2 });
    });
    const initialElapsed = result.current.snapshot?.state.elapsedMilliseconds ?? 0;

    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(result.current.snapshot?.state.elapsedMilliseconds).toBeGreaterThan(
      initialElapsed,
    );
  });
});
