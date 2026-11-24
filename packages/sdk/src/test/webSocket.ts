import { vi } from "vitest";

type SocketHandler<T> = ((event: T) => void) | null;

export class TestWebSocket {
  public static readonly CONNECTING = 0;
  public static readonly OPEN = 1;
  public static readonly CLOSING = 2;
  public static readonly CLOSED = 3;
  private static readonly sockets: TestWebSocket[] = [];

  public readonly url: string;
  public readyState = TestWebSocket.CONNECTING;
  public sent: string[] = [];
  public onopen: SocketHandler<Event> = null;
  public onmessage: SocketHandler<MessageEvent> = null;
  public onerror: SocketHandler<Event> = null;
  public onclose: SocketHandler<CloseEvent> = null;

  public constructor(url: string) {
    this.url = url;
    TestWebSocket.sockets.push(this);
  }

  public static latest(): TestWebSocket {
    const socket = TestWebSocket.sockets.at(-1);
    if (!socket) throw new Error("No websocket has been created");
    return socket;
  }

  public static reset(): void {
    TestWebSocket.sockets.length = 0;
  }

  public open(): void {
    this.readyState = TestWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  public message(data: unknown): void {
    this.onmessage?.(new MessageEvent("message", { data }));
  }

  public fail(): void {
    this.onerror?.(new Event("error"));
  }

  public serverClose(): void {
    this.readyState = TestWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  public send(payload: string): void {
    this.sent.push(payload);
  }

  public close(): void {
    this.readyState = TestWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }
}

export const installWebSocketMock = (): void => {
  TestWebSocket.reset();
  vi.stubGlobal("WebSocket", TestWebSocket);
};
