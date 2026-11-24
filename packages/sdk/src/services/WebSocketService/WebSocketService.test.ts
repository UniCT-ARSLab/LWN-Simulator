import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WebSocketService } from "./WebSocketService";
import { installWebSocketMock, TestWebSocket } from "../../test/webSocket";

describe("WebSocketService", () => {
  beforeEach(() => installWebSocketMock());

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("derives the websocket endpoint and publishes connection state", () => {
    const service = new WebSocketService("https://api.example.test/", 10);
    const states: string[] = [];
    service.onStateChange((state) => states.push(state));

    service.connect("/ws");
    const socket = TestWebSocket.latest();

    expect(socket.url).toBe("wss://api.example.test/ws");
    expect(states).toEqual(["disconnected", "connecting"]);

    socket.open();

    expect(service.connectionState).toBe("connected");
    expect(states).toEqual(["disconnected", "connecting", "connected"]);
  });

  it("delivers messages, serializes JSON, and reports socket errors", () => {
    const service = new WebSocketService("http://localhost:8000");
    const messages: string[] = [];
    const errors: Array<Error | null> = [];
    service.subscribe((message) => messages.push(message));
    service.onError((error) => errors.push(error));

    service.connect("ws");
    const socket = TestWebSocket.latest();
    socket.open();
    socket.message({ event: "device.created" });
    service.sendJSON({ action: "ping" });
    socket.fail();

    expect(messages).toEqual(["[object Object]"]);
    expect(socket.sent).toEqual(['{"action":"ping"}']);
    expect(errors.at(-1)?.message).toBe("websocket connection failed");
  });

  it("reconnects after an unexpected close and stops after disconnect", () => {
    vi.useFakeTimers();
    const service = new WebSocketService("http://localhost:8000", 25);
    service.connect("/ws");
    const firstSocket = TestWebSocket.latest();
    firstSocket.open();
    firstSocket.serverClose();

    expect(service.connectionState).toBe("reconnecting");
    vi.advanceTimersByTime(25);

    const secondSocket = TestWebSocket.latest();
    expect(secondSocket).not.toBe(firstSocket);
    expect(secondSocket.readyState).toBe(TestWebSocket.CONNECTING);

    service.disconnect();
    expect(service.connectionState).toBe("disconnected");
    vi.advanceTimersByTime(100);
    expect(TestWebSocket.latest()).toBe(secondSocket);
  });

  it("rejects sends when no connection is open", () => {
    const service = new WebSocketService("http://localhost:8000");

    expect(() => service.send("payload")).toThrow("websocket is not connected");
  });
});
