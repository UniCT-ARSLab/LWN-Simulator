import {
  Device,
  SimulationActionResponse,
  SimulationDownlinkRequest,
  SimulationMACCommand,
  SimulationMACCommandRequest,
  SimulationMACCommandType,
  SimulationUplinkRequest,
} from "@lwn-simulator/contracts";

/** Command console tab currently selected by the user. */
export enum SimulationCommandMode {
  /** Sends an uplink from the selected device. */
  Uplink = "uplink",
  /** Queues a downlink for the selected device. */
  Downlink = "downlink",
  /** Queues a MAC command for the selected device. */
  MACCommand = "mac-command",
}

/** Names of configurable MAC command parameters. */
export type SimulationMACCommandFieldName = Exclude<
  keyof SimulationMACCommand,
  "type"
>;

/** Metadata used to render one MAC command field. */
export interface SimulationMACCommandFieldDescriptor {
  /** Contract property represented by the field. */
  name: SimulationMACCommandFieldName;
  /** Human-readable field label. */
  label: string;
  /** Explanatory text shown with the field. */
  info: string;
  /** Form control type used to edit the value. */
  input: "number" | "checkbox";
  /** Whether the field must be provided. */
  required?: boolean;
  /** Inclusive minimum numeric value. */
  minimum?: number;
  /** Inclusive maximum numeric value. */
  maximum?: number;
  /** Numeric increment used by the field. */
  step?: number;
  /** Placeholder shown before a value is entered. */
  placeholder?: string;
}

/** MAC command field descriptors indexed by command type. */
export type SimulationMACCommandFields = Record<
  SimulationMACCommandType,
  Array<SimulationMACCommandFieldDescriptor>
>;

/** Properties required by the simulation command console. */
export interface SimulationCommandPanelProps {
  /** Devices that can receive or transmit injected traffic. */
  devices: Array<Device>;
  /** Whether command submission is currently allowed. */
  enabled: boolean;
  /** Queues an uplink request. */
  onQueueUplink: (
    request: SimulationUplinkRequest,
  ) => Promise<SimulationActionResponse>;
  /** Queues a downlink request. */
  onQueueDownlink: (
    request: SimulationDownlinkRequest,
  ) => Promise<SimulationActionResponse>;
  /** Queues a MAC command request. */
  onQueueMACCommand: (
    request: SimulationMACCommandRequest,
  ) => Promise<SimulationActionResponse>;
}
