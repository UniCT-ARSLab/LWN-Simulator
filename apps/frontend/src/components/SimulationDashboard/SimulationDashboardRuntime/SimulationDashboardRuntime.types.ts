import { RadioProtocol, RuntimeDevice, RuntimeGateway } from "@lwn-simulator/contracts";

/** Properties required to render runtime device and gateway lists. */
export interface SimulationDashboardRuntimeProps {
  /** Runtime state for devices in the current simulation. */
  devices: Array<RuntimeDevice>;
  /** Transmit protocol of each configured device, indexed by its internal identifier. */
  deviceProtocols?: Readonly<Record<string, RadioProtocol | undefined>>;
  /** Runtime state for gateways in the current simulation. */
  gateways: Array<RuntimeGateway>;
  /** Converts a gateway state enum into readable text. */
  formatState(value: string): string;
}
