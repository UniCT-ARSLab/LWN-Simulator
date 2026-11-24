import { SimulationLogObject } from "@lwn-simulator/contracts";

/** Properties used by the virtualized simulation-object list. */
export interface VirtualObjectListProps {
  /** Devices and gateways available in the selected simulation run. */
  objects: Array<SimulationLogObject>;
  /** Object whose event history is currently displayed. */
  selectedObject: SimulationLogObject | null;
  /** Selects an object and loads its event history. */
  onSelect(object: SimulationLogObject): void;
}
