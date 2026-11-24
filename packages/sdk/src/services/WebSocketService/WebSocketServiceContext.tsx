import { createContext } from "react";
import { WebSocketServiceContent } from "./WebSocketService.types";

const WebSocketServiceContext =
  createContext<WebSocketServiceContent | null>(null);

export default WebSocketServiceContext;
