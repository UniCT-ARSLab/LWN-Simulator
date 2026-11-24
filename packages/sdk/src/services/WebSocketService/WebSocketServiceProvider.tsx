import { useCallback, useEffect, useMemo, useState } from "react";
import WebSocketServiceContext from "./WebSocketServiceContext";
import { WebSocketService } from "./WebSocketService";
import {
  WebSocketServiceProviderProps,
  WebSocketMessageListener,
  WebSocketServiceContent,
  WebSocketConnectionState,
} from "./WebSocketService.types";

const WebSocketServiceProvider = (props: WebSocketServiceProviderProps) => {
  // States
  const [connectionState, setConnectionState] =
    useState<WebSocketConnectionState>("disconnected");
  const [error, setError] = useState<Error | null>(null);

  // Memos
  const client = useMemo(
    () => new WebSocketService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const connect = useCallback((path: string) => client.connect(path), [client]);
  const disconnect = useCallback(() => client.disconnect(), [client]);
  const send = useCallback((payload: string) => client.send(payload), [client]);
  const sendJSON = useCallback(
    (payload: unknown) => client.sendJSON(payload),
    [client],
  );
  const subscribe = useCallback(
    (listener: WebSocketMessageListener) => client.subscribe(listener),
    [client],
  );

  // Effects

  useEffect(() => {
    const removeStateListener = client.onStateChange(setConnectionState);
    const removeErrorListener = client.onError(setError);

    return () => {
      removeStateListener();
      removeErrorListener();
      client.dispose();
    };
  }, [client]);

  useEffect(() => {
    client.connect("/ws");
    return () => client.disconnect();
  }, [client]);

  const value = useMemo(
    (): WebSocketServiceContent => ({
      connect,
      disconnect,
      send,
      sendJSON,
      subscribe,
      connectionState,
      error,
    }),
    [connect, disconnect, send, sendJSON, subscribe, connectionState, error],
  );

  return (
    <WebSocketServiceContext.Provider value={value}>
      {props.children}
    </WebSocketServiceContext.Provider>
  );
};

export default WebSocketServiceProvider;
