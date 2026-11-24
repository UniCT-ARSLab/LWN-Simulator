import { act, renderHook, waitFor } from "@testing-library/react";
import type { Gateway } from "@lwn-simulator/contracts";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiCaller } from "../../models/ApiCaller";
import { installWebSocketMock, TestWebSocket } from "../../test/webSocket";
import WebSocketServiceProvider from "../WebSocketService/WebSocketServiceProvider";
import GatewayServiceProvider from "./GatewayServiceProvider";
import { useGatewayService } from "./useGatewayService";

const gateway = {
  id: "gateway-1",
  name: "Catania Gateway",
} as Gateway;

const wrapper = ({ children }: PropsWithChildren) => (
  <WebSocketServiceProvider baseUrl="http://localhost:8000">
    <GatewayServiceProvider baseUrl="http://localhost:8000" profileID="profile-1">
      {children}
    </GatewayServiceProvider>
  </WebSocketServiceProvider>
);

describe("GatewayServiceProvider", () => {
  beforeEach(() => installWebSocketMock());

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("does not duplicate a gateway received from the realtime event after creation", async () => {
    const getSpy = vi.spyOn(ApiCaller.prototype, "get").mockResolvedValue({
      gateways: [],
    } as never);
    const postSpy = vi.spyOn(ApiCaller.prototype, "post").mockResolvedValue({
      gateway,
    } as never);
    const { result } = renderHook(() => useGatewayService(), { wrapper });
    TestWebSocket.latest().open();
    await waitFor(() => expect(getSpy).toHaveBeenCalled());

    await act(async () => {
      await result.current.createGateway({} as never);
    });
    expect(result.current.gateways).toEqual([gateway]);
    expect(postSpy).toHaveBeenCalledWith("/create-gateway", {});

    act(() =>
      TestWebSocket.latest().message(
        JSON.stringify({
          type: "gateway.created",
          timestampMilliseconds: 2,
          profileID: "profile-1",
          gateway,
        }),
      ),
    );
    await waitFor(() => expect(result.current.gateways).toEqual([gateway]));
  });
});
