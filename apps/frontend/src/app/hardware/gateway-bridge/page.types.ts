import { FormLogic, FormValue } from "@lwn-simulator/ui-components";

/** Values persisted for the UDP gateway bridge configuration. */
export interface GatewayBridgeFormValues {
  /** Enables bridge traffic for the next simulation. */
  enabled: boolean;
  /** Host name or IPv4 address receiving bridge packets. */
  address: string;
  /** UDP port receiving bridge packets. */
  port: number;
}

/** Runtime state exposed by the gateway bridge settings form. */
export interface GatewayBridgeFormState {
  /** Current form logic, when the form has been mounted. */
  logic: FormLogic | null;
  /** Values currently shown by the form. */
  values: GatewayBridgeFormValues;
  /** Last configuration successfully persisted by the user. */
  lastSaved: GatewayBridgeFormValues | null;
}

/** Map shape used when submitting gateway bridge form values. */
export type GatewayBridgeFormValueMap = Record<string, FormValue>;
