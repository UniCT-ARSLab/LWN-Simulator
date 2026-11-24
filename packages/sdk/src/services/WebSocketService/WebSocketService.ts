import { ApiCaller } from "../../models/ApiCaller";

import {
  WebSocketConnectionState,
  WebSocketMessageListener,
  WebSocketStateListener,
} from "./WebSocketService.types";

const defaultReconnectDelayMilliseconds = 1000;

export class WebSocketService {
  private readonly messageListeners = new Set<WebSocketMessageListener>();
  private readonly stateListeners = new Set<WebSocketStateListener>();
  private readonly errorListeners = new Set<(error: Error | null) => void>();
  private readonly reconnectDelayMilliseconds: number;
  private socket: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private connectionPath: string | null = null;
  private state: WebSocketConnectionState = "disconnected";
  private error: Error | null = null;
  private shouldReconnect = false;
  private generation = 0;

  public constructor(
    private readonly baseUrl: string,
    reconnectDelayMilliseconds = defaultReconnectDelayMilliseconds,
  ) {
    this.reconnectDelayMilliseconds = reconnectDelayMilliseconds;
  }

  public get connectionState(): WebSocketConnectionState {
    return this.state;
  }

  public get lastError(): Error | null {
    return this.error;
  }

  public connect(path: string): void {
    if (
      this.connectionPath === path &&
      this.socket &&
      (this.socket.readyState === WebSocket.OPEN ||
        this.socket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    this.disconnect(false);
    this.connectionPath = path;
    this.shouldReconnect = true;
    this.open(false);
  }

  public disconnect(clearPath = true): void {
    this.shouldReconnect = false;
    this.clearReconnectTimer();
    this.generation += 1;
    this.socket?.close();
    this.socket = null;
    if (clearPath) this.connectionPath = null;
    this.setError(null);
    this.setState("disconnected");
  }

  public dispose(): void {
    this.disconnect();
    this.messageListeners.clear();
    this.stateListeners.clear();
    this.errorListeners.clear();
  }

  public subscribe(listener: WebSocketMessageListener): () => void {
    this.messageListeners.add(listener);
    return () => this.messageListeners.delete(listener);
  }

  public onStateChange(listener: WebSocketStateListener): () => void {
    this.stateListeners.add(listener);
    listener(this.state);
    return () => this.stateListeners.delete(listener);
  }

  public onError(listener: (error: Error | null) => void): () => void {
    this.errorListeners.add(listener);
    listener(this.error);
    return () => this.errorListeners.delete(listener);
  }

  public send(payload: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error("websocket is not connected");
    }
    this.socket.send(payload);
  }

  public sendJSON(payload: unknown): void {
    this.send(JSON.stringify(payload));
  }

  private open(isReconnect: boolean): void {
    if (!this.connectionPath || !this.shouldReconnect) return;

    const generation = ++this.generation;
    this.setState(isReconnect ? "reconnecting" : "connecting");
    const socket = new WebSocket(this.getWebSocketUrl(this.connectionPath));
    this.socket = socket;

    socket.onopen = () => {
      if (generation !== this.generation) return;
      this.setError(null);
      this.setState("connected");
    };

    socket.onmessage = (event) => {
      if (generation !== this.generation) return;
      const data =
        typeof event.data === "string" ? event.data : String(event.data);
      this.messageListeners.forEach((listener) => listener(data));
    };

    socket.onerror = () => {
      if (generation !== this.generation) return;
      this.setError(new Error("websocket connection failed"));
    };

    socket.onclose = () => {
      if (generation !== this.generation) return;
      this.socket = null;
      if (this.shouldReconnect && this.connectionPath) {
        this.scheduleReconnect();
      } else {
        this.setState("disconnected");
      }
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer || !this.shouldReconnect) return;
    this.setState("reconnecting");
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.open(true);
    }, this.reconnectDelayMilliseconds);
  }

  private clearReconnectTimer(): void {
    if (!this.reconnectTimer) return;
    clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
  }

  private getWebSocketUrl(path: string): string {
    const url = new URL(ApiCaller.joinUrl(this.baseUrl, path));
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    return url.toString();
  }

  private setState(state: WebSocketConnectionState): void {
    if (this.state === state) return;
    this.state = state;
    this.stateListeners.forEach((listener) => listener(state));
  }

  private setError(error: Error | null): void {
    this.error = error;
    this.errorListeners.forEach((listener) => listener(error));
  }
}
