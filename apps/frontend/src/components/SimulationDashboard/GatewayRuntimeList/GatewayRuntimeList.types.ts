import { RuntimeGateway } from "@lwn-simulator/contracts";

/** Properties accepted by the runtime gateway list. */
export interface GatewayRuntimeListProps {
  /** Runtime gateway connections displayed by the list. */
  gateways: Array<RuntimeGateway>;
  /** Converts a gateway state into readable text. */
  formatState(value: string): string;
}
