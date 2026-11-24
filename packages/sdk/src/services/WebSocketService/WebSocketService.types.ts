import { PropsWithChildren } from "react";

/** State of the shared websocket connection. */
export type WebSocketConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

/** Receives a raw message from the shared websocket connection. */
export type WebSocketMessageListener = (message: string) => void;

/** Receives connection state changes from the shared websocket connection. */
export type WebSocketStateListener = (
  state: WebSocketConnectionState,
) => void;

/** Public operations and state exposed by the websocket service context. */
export interface WebSocketServiceContent {
  /** Opens or reuses a websocket connection for the supplied path. */
  connect: (path: string) => void;

  /** Closes the current connection and cancels reconnect attempts. */
  disconnect: () => void;

  /** Sends a raw string payload through the active connection. */
  send: (payload: string) => void;

  /** Serializes and sends a payload through the active connection. */
  sendJSON: (payload: unknown) => void;

  /** Subscribes to raw messages and returns an unsubscribe callback. */
  subscribe: (listener: WebSocketMessageListener) => () => void;

  /** Current websocket connection state. */
  connectionState: WebSocketConnectionState;

  /** Most recent connection error, if any. */
  error: Error | null;
}

/** Configuration required by the websocket service provider. */
export interface WebSocketServiceProviderProps extends PropsWithChildren {
  /** Base HTTP(S) URL used to derive the websocket endpoint. */
  baseUrl: string;
}
