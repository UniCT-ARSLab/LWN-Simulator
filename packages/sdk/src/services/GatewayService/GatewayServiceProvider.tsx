import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  CreateGatewayRequest,
  DeleteGatewayRequest,
  Gateway,
  GetGatewayRequest,
  RealtimeWebSocketMessage,
  UpdateGatewayRequest,
} from "@lwn-simulator/contracts";
import GatewayServiceContext from "./GatewayServiceContext";
import { GatewayService } from "./GatewayService";
import {
  GatewayServiceContent,
  GatewayServiceProviderProps,
} from "./GatewayService.types";
import { useWebSocketService } from "../WebSocketService";
import { RealtimeWebSocketMessageType } from "../../models/RealtimeWebSocket";

const GatewayServiceProvider = (props: GatewayServiceProviderProps) => {
  // States
  const [gateways, setGateways] = useState<Array<Gateway> | null>(null);
  const [error, setError] = useState<Error | null>(null);

  // Hooks
  const { connectionState, subscribe } = useWebSocketService();
  const hasConnectedRef = useRef(false);

  // Memos
  const service = useMemo(
    () => new GatewayService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const createGateway = useCallback(
    async (req: CreateGatewayRequest): Promise<Gateway> => {
      const res = await service.createGateway(req);
      setGateways((previous) => {
        if (!previous) return [res];
        const existingIndex = previous.findIndex(
          (gateway) => gateway.id === res.id,
        );
        if (existingIndex === -1) return [...previous, res];

        return previous.map((gateway, index) =>
          index === existingIndex ? res : gateway,
        );
      });
      setError(null);
      return res;
    },
    [service],
  );

  const getGateway = useCallback(
    (req: GetGatewayRequest): Promise<Gateway> => service.getGateway(req),
    [service],
  );

  const getGateways = useCallback(async (): Promise<Array<Gateway>> => {
    const res = await service.getGateways();
    setGateways(res);
    setError(null);
    return res;
  }, [service]);

  const updateGateway = useCallback(
    async (req: UpdateGatewayRequest): Promise<Gateway> => {
      const res = await service.updateGateway(req);
      setGateways(
        (previous) =>
          previous?.map((gateway) =>
            gateway.id === res.id ? res : gateway,
          ) ?? null,
      );
      setError(null);
      return res;
    },
    [service],
  );

  const deleteGateway = useCallback(
    async (req: DeleteGatewayRequest): Promise<void> => {
      await service.deleteGateway(req);
      setGateways(
        (previous) =>
          previous?.filter((gateway) => gateway.id !== req.id) ?? null,
      );
      setError(null);
    },
    [service],
  );

  const handleRealtimeMessage = useCallback((rawMessage: string) => {
    try {
      const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
      if (message.profileID && message.profileID !== props.profileID) return;
      if (
        (message.type === RealtimeWebSocketMessageType.GatewayCreated ||
          message.type === RealtimeWebSocketMessageType.GatewayUpdated) &&
        message.gateway
      ) {
        setGateways((previous) => {
          if (!previous) return [message.gateway as Gateway];
          const exists = previous.some(
            (gateway) => gateway.id === message.gateway?.id,
          );
          return exists
            ? previous.map((gateway) =>
                gateway.id === message.gateway?.id
                  ? (message.gateway as Gateway)
                  : gateway,
              )
            : [...previous, message.gateway as Gateway];
        });
      } else if (
        message.type === RealtimeWebSocketMessageType.GatewayDeleted &&
        message.resourceID
      ) {
        setGateways((previous) =>
          previous?.filter((gateway) => gateway.id !== message.resourceID) ??
          null,
        );
      }
    } catch {
      return;
    }
  }, [props.profileID]);

  useEffect(
    () => subscribe(handleRealtimeMessage),
    [handleRealtimeMessage, props.profileID, subscribe],
  );

  useEffect(() => {
    if (connectionState !== "connected") return;
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true;
      return;
    }

    void getGateways().catch(() => {
      setError(
        new Error("Failed to recover gateways after websocket reconnect"),
      );
    });
  }, [connectionState, getGateways]);

  useEffect(() => {
    void Promise.resolve()
      .then(getGateways)
      .catch((err) => {
        setGateways([]);
        setError(
          err instanceof Error ? err : new Error("Failed to load gateways"),
        );
      });
  }, [getGateways]);

  // Memos
  const value = useMemo(
    (): GatewayServiceContent => ({
      createGateway,
      getGateway,
      getGateways,
      updateGateway,
      deleteGateway,
      gateways,
      error,
    }),
    [
      createGateway,
      getGateway,
      getGateways,
      updateGateway,
      deleteGateway,
      gateways,
      error,
    ],
  );

  return (
    <GatewayServiceContext.Provider value={value}>
      {props.children}
    </GatewayServiceContext.Provider>
  );
};

export default GatewayServiceProvider;
