import { afterEach, describe, expect, it } from "vitest";
import {
  installWebSocketMock,
  mockWebSocket,
} from "./mocks/webSocket";

describe("frontend test infrastructure", () => {
  let restoreWebSocket: (() => void) | undefined;

  afterEach(() => {
    restoreWebSocket?.();
    restoreWebSocket = undefined;
  });

  it("provides a controllable WebSocket boundary", () => {
    restoreWebSocket = installWebSocketMock();

    const socket = new WebSocket("ws://test.local/api/ws");
    socket.onopen = () => undefined;
    mockWebSocket.latest().open();
    socket.send("hello");

    expect(socket.readyState).toBe(WebSocket.OPEN);
    expect(mockWebSocket.latest().sentMessages).toEqual(["hello"]);
  });
});
