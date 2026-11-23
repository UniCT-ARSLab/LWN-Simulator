import {
  SimulationActionResponse,
  SimulationConfig,
  SimulationDownlinkRequest,
  SimulationEvent,
  SimulationEventsResponse,
  SimulationRun,
  SimulationRunsResponse,
  SimulationMACCommandRequest,
  SimulationSnapshot,
  SimulationSpeedRequest,
  SimulationUplinkRequest,
} from "@lwn-simulator/contracts";
import { PropsWithChildren } from "react";
import { WebSocketConnectionState } from "../WebSocketService";

/** Connection state used by the simulation realtime stream. */
export type SimulationConnectionState = WebSocketConnectionState;

/** Filters and pagination options for simulation event queries. */
export interface GetSimulationEventsOptions {
  /** Return events after this sequence number. */
  afterSequence?: number;

  /** Return events before this sequence number. */
  beforeSequence?: number;

  /** Maximum number of events to return. */
  limit?: number;

  /** Restrict results to a device. */
  deviceID?: string;

  /** Restrict results to a gateway. */
  gatewayID?: string;

  /** Restrict results to an event type. */
  type?: string;
}

/** Operations, realtime state, and cached data exposed by the simulation service. */
export interface SimulationServiceContent {
  /** Starts a simulation. */
  start: (config: SimulationConfig) => Promise<SimulationSnapshot>;

  /** Changes the simulation speed. */
  setSpeed: (request: SimulationSpeedRequest) => Promise<SimulationSnapshot>;

  /** Pauses the active simulation. */
  pause: () => Promise<SimulationSnapshot>;

  /** Resumes a paused simulation. */
  resume: () => Promise<SimulationSnapshot>;

  /** Stops the active simulation. */
  stop: () => Promise<SimulationSnapshot>;

  /** Retrieves the current simulation snapshot. */
  getSnapshot: () => Promise<SimulationSnapshot>;

  /** Queues an uplink. */
  queueUplink: (
    req: SimulationUplinkRequest,
  ) => Promise<SimulationActionResponse>;

  /** Queues a downlink. */
  queueDownlink: (
    req: SimulationDownlinkRequest,
  ) => Promise<SimulationActionResponse>;

  /** Queues a MAC command. */
  queueMACCommand: (
    req: SimulationMACCommandRequest,
  ) => Promise<SimulationActionResponse>;

  /** Retrieves simulation events. */
  getEvents: (
    options?: GetSimulationEventsOptions,
  ) => Promise<SimulationEventsResponse>;

  /** Builds the profile-scoped JSON download URL for a run, or the ZIP URL for all runs. */
  getLogExportURL: (id?: string) => string;

  /** Retrieves stored simulation runs. */
  getLogs: () => Promise<SimulationRunsResponse>;

  /** Retrieves one stored simulation run. */
  getLog: (id: string) => Promise<SimulationRun>;

  /** Retrieves events belonging to one stored simulation run. */
  getLogEvents: (
    id: string,
    options?: GetSimulationEventsOptions,
  ) => Promise<SimulationEventsResponse>;

  /** Retrieves Prometheus-formatted simulation metrics. */
  getMetrics: () => Promise<string>;

  /** Starts the realtime simulation connection. */
  connect: () => void;

  /** Stops local realtime processing. */
  disconnect: () => void;

  /** Latest simulation snapshot received from the backend. */
  snapshot: SimulationSnapshot | null;

  /** Cached simulation events received from the backend. */
  events: Array<SimulationEvent>;

  /** Current realtime connection state. */
  connectionState: SimulationConnectionState;

  /** Last simulation or realtime error. */
  error: Error | null;
}

/** Properties accepted by the simulation service provider. */
export interface SimulationServiceProviderProps extends PropsWithChildren {
  /** Base URL of the backend API. */
  baseUrl: string;

  /** Profile identifier used to filter realtime simulation events. */
  profileID?: string;

  /** Reports the status returned by a successful simulation control request. */
  onControlStatus?: (profileID: string | undefined, status: SimulationSnapshot["state"]["status"]) => void;
}
