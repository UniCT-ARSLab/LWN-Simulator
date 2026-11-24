/**
 * Runtime names used by the websocket protocol.
 *
 * The SDK keeps these values locally so its CommonJS build does not require
 * the TypeScript-only contracts package at runtime.
 */
export const RealtimeWebSocketMessageType = {
  ProfileCreated: "profile.created",
  ProfileUpdated: "profile.updated",
  ProfileDeleted: "profile.deleted",
  ProfileGatewayBridgeUpdated: "profile.gateway-bridge.updated",
  DeviceCreated: "device.created",
  DeviceUpdated: "device.updated",
  DeviceDeleted: "device.deleted",
  GatewayCreated: "gateway.created",
  GatewayUpdated: "gateway.updated",
  GatewayDeleted: "gateway.deleted",
  SimulationSnapshot: "simulation.snapshot",
  SimulationEvent: "simulation.event",
  SimulationRunCreated: "simulation.run.created",
  SimulationRunUpdated: "simulation.run.updated",
  Error: "error",
} as const;

/**
 * Runtime simulation states used when determining whether a simulation is
 * currently active.
 */
export const SimulationStatus = {
  Idle: "idle",
  Starting: "starting",
  Running: "running",
  Pausing: "pausing",
  Paused: "paused",
  Resuming: "resuming",
  Stopping: "stopping",
  Stopped: "stopped",
  Recovering: "recovering",
  Failed: "failed",
} as const;
