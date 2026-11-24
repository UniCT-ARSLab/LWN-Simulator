export type MockWebSocketMessage = string | ArrayBuffer | Blob;

export type MockWebSocketInstance = {
  readonly url: string;
  readonly sentMessages: string[];
  readonly readyState: number;
  open: () => void;
  receive: (message: MockWebSocketMessage) => void;
  fail: () => void;
  close: () => void;
};

type MockSocketEventHandler = ((event: Event) => void) | null;
type MockSocketMessageHandler = ((event: MessageEvent) => void) | null;

const CONNECTING = 0;
const OPEN = 1;
const CLOSING = 2;
const CLOSED = 3;

class TestWebSocket {
  public static readonly CONNECTING = CONNECTING;
  public static readonly OPEN = OPEN;
  public static readonly CLOSING = CLOSING;
  public static readonly CLOSED = CLOSED;
  public binaryType: BinaryType = "blob";
  public bufferedAmount = 0;
  public extensions = "";
  public onclose: MockSocketEventHandler = null;
  public onerror: MockSocketEventHandler = null;
  public onmessage: MockSocketMessageHandler = null;
  public onopen: MockSocketEventHandler = null;
  public protocol = "";
  public readonly sentMessages: string[] = [];
  public readyState = CONNECTING;
  public url: string;

  public constructor(url: string) {
    this.url = url;
    TestWebSocket.instances.push(this);
  }

  public static instances: TestWebSocket[] = [];

  public static reset(): void {
    TestWebSocket.instances.forEach((socket) => socket.close());
    TestWebSocket.instances = [];
  }

  public static latest(): MockWebSocketInstance {
    const socket = TestWebSocket.instances.at(-1);
    if (!socket) {
      throw new Error("No mock WebSocket instance exists");
    }
    return socket;
  }

  public open(): void {
    this.readyState = OPEN;
    this.onopen?.(new Event("open"));
  }

  public receive(message: MockWebSocketMessage): void {
    this.onmessage?.(new MessageEvent("message", { data: message }));
  }

  public fail(): void {
    this.onerror?.(new Event("error"));
  }

  public close(): void {
    if (this.readyState === CLOSED) {
      return;
    }
    this.readyState = CLOSING;
    this.onclose?.(new Event("close"));
    this.readyState = CLOSED;
  }

  public send(message: string): void {
    if (this.readyState !== OPEN) {
      throw new Error("Mock WebSocket is not open");
    }
    this.sentMessages.push(message);
  }

  public addEventListener(
    _type: string,
    _listener: EventListenerOrEventListenerObject | null,
  ): void {}

  public removeEventListener(
    _type: string,
    _listener: EventListenerOrEventListenerObject | null,
  ): void {}

  public dispatchEvent(_event: Event): boolean {
    return true;
  }

}

export const installWebSocketMock = (): (() => void) => {
  const originalDescriptor = Object.getOwnPropertyDescriptor(
    globalThis,
    "WebSocket",
  );
  TestWebSocket.reset();
  Object.defineProperty(globalThis, "WebSocket", {
    configurable: true,
    value: TestWebSocket,
    writable: true,
  });

  return () => {
    TestWebSocket.reset();
    if (originalDescriptor) {
      Object.defineProperty(globalThis, "WebSocket", originalDescriptor);
    } else {
      delete (globalThis as { WebSocket?: unknown }).WebSocket;
    }
  };
};

export const mockWebSocket = {
  latest: (): MockWebSocketInstance => TestWebSocket.latest(),
  reset: (): void => TestWebSocket.reset(),
};
