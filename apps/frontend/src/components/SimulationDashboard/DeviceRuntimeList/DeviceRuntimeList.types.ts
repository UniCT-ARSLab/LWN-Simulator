import { RadioProtocol, RuntimeDevice } from "@lwn-simulator/contracts";

/** Properties accepted by the runtime device list. */
export interface DeviceRuntimeListProps {
  /** Runtime device sessions displayed by the list. */
  devices: Array<RuntimeDevice>;
  /** Transmit protocol of each configured device, indexed by its internal identifier. */
  deviceProtocols?: Readonly<Record<string, RadioProtocol | undefined>>;
}
