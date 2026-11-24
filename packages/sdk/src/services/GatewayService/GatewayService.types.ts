import {
  CreateGatewayRequest,
  DeleteGatewayRequest,
  Gateway,
  GetGatewayRequest,
  UpdateGatewayRequest,
} from "@lwn-simulator/contracts";
import { PropsWithChildren } from "react";

/** Operations and state exposed by the gateway service provider. */
export interface GatewayServiceContent {
  /** Creates a gateway. */
  createGateway: (req: CreateGatewayRequest) => Promise<Gateway>;

  /** Retrieves a gateway by identifier. */
  getGateway: (req: GetGatewayRequest) => Promise<Gateway>;

  /** Retrieves all gateways in the active profile. */
  getGateways: () => Promise<Array<Gateway>>;

  /** Updates an existing gateway. */
  updateGateway: (req: UpdateGatewayRequest) => Promise<Gateway>;

  /** Deletes a gateway. */
  deleteGateway: (req: DeleteGatewayRequest) => Promise<void>;

  /** Gateways loaded from the backend, or `null` before the first load. */
  gateways: Array<Gateway> | null;

  /** Last loading or mutation error. */
  error: Error | null;
}

/** Properties accepted by the gateway service provider. */
export interface GatewayServiceProviderProps extends PropsWithChildren {
  /** Base URL of the backend API. */
  baseUrl: string;

  /** Profile identifier used to filter realtime gateway events. */
  profileID?: string;
}
