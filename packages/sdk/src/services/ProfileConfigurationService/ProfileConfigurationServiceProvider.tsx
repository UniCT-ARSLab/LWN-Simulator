"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { GatewayBridgeConfig, RealtimeWebSocketMessage } from "@lwn-simulator/contracts";
import { useWebSocketService } from "../WebSocketService";
import { ProfileConfigurationService } from "./ProfileConfigurationService";
import ProfileConfigurationServiceContext from "./ProfileConfigurationServiceContext";
import { ProfileConfigurationServiceProviderProps } from "./ProfileConfigurationService.types";
import { RealtimeWebSocketMessageType } from "../../models/RealtimeWebSocket";

const ProfileConfigurationServiceProvider = (
  props: ProfileConfigurationServiceProviderProps,
) => {
  // States
  const [gatewayBridgeConfig, setGatewayBridgeConfig] =
    useState<GatewayBridgeConfig | null>(null);

  // Hooks
  const { subscribe } = useWebSocketService();

  // Memos
  const service = useMemo(
    () => new ProfileConfigurationService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const getGatewayBridge = useCallback(async () => {
    const config = await service.getGatewayBridge();
    setGatewayBridgeConfig(config);
    return config;
  }, [service]);

  const updateGatewayBridge = useCallback(
    async (config: GatewayBridgeConfig) => {
      const updated = await service.updateGatewayBridge(config);
      setGatewayBridgeConfig(updated);
      return updated;
    },
    [service],
  );

  const handleRealtimeMessage = useCallback(
    (rawMessage: string) => {
      try {
        const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
        if (message.profileID && message.profileID !== props.profileID) return;
        if (
          message.type ===
            RealtimeWebSocketMessageType.ProfileGatewayBridgeUpdated &&
          message.gatewayBridge
        ) {
          setGatewayBridgeConfig(message.gatewayBridge);
        }
      } catch {
        return;
      }
    },
    [props.profileID],
  );

  // Effects
  useEffect(() => {
    void Promise.resolve()
      .then(getGatewayBridge)
      .catch(() => setGatewayBridgeConfig(null));
  }, [getGatewayBridge]);

  useEffect(
    () => subscribe(handleRealtimeMessage),
    [handleRealtimeMessage, subscribe],
  );

  // Memos
  const value = useMemo(
    () => ({
      gatewayBridgeConfig,
      getGatewayBridge,
      updateGatewayBridge,
    }),
    [gatewayBridgeConfig, getGatewayBridge, updateGatewayBridge],
  );

  return (
    <ProfileConfigurationServiceContext.Provider value={value}>
      {props.children}
    </ProfileConfigurationServiceContext.Provider>
  );
};

export default ProfileConfigurationServiceProvider;
