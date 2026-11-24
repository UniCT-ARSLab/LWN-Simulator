import { FormField, FormValue } from "@lwn-simulator/ui-components";
import { SimulationStatus } from "@lwn-simulator/contracts";

/** Properties required to render the simulation controls. */
export interface SimulationDashboardControlsProps {
  /** Fields used to configure a new simulation or update its speed. */
  fields: Array<FormField>;
  /** Current engine status. */
  status: SimulationStatus;
  /** Whether the current profile can start a simulation. */
  canStart: boolean;
  /** Whether external gateway connections require the real-time simulation clock. */
  requiresRealtime?: boolean;
  /** Whether the device and gateway collections have finished loading. */
  hardwareLoaded: boolean;
  /** Whether the current profile contains an active device or gateway. */
  hasActiveHardware: boolean;
  /** Whether another profile owns the active simulation. */
  activeSimulationIsElsewhere: boolean;
  /** Optional name of the profile currently running the simulation. */
  activeSimulationProfileName?: string;
  /** Starts a simulation with the submitted form values. */
  onStart(values: Record<string, FormValue>): void | Promise<void>;
  /** Pauses the current simulation. */
  onPause(): void | Promise<void>;
  /** Resumes the current simulation. */
  onResume(): void | Promise<void>;
  /** Stops the current simulation. */
  onStop(): void | Promise<void>;
}
