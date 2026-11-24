import { GatewayBridgeConfig } from "@lwn-simulator/contracts";
import { PropsWithChildren } from "react";

/** Operations and state exposed by the profile configuration service. */
export interface ProfileConfigurationServiceContent {
  /** Current gateway bridge configuration, or `null` before loading. */
  gatewayBridgeConfig: GatewayBridgeConfig | null;

  /** Retrieves the gateway bridge configuration. */
  getGatewayBridge: () => Promise<GatewayBridgeConfig>;

  /** Saves and returns the gateway bridge configuration. */
  updateGatewayBridge: (
    config: GatewayBridgeConfig,
  ) => Promise<GatewayBridgeConfig>;
}

/** Properties accepted by the profile configuration service provider. */
export interface ProfileConfigurationServiceProviderProps
  extends PropsWithChildren {
  /** Base URL of the backend API. */
  baseUrl: string;

  /** Profile identifier whose configuration is managed. */
  profileID: string;
}
