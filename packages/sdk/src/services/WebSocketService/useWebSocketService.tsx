import { useContext } from "react";
import WebSocketServiceContext from "./WebSocketServiceContext";

export const useWebSocketService = () => {
  const context = useContext(WebSocketServiceContext);

  if (!context) {
    throw new Error(
      "useWebSocketService must be used inside `WebSocketServiceProvider`",
    );
  }

  return context;
};
