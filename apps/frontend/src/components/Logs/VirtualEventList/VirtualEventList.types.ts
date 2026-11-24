import { SimulationEvent } from "@lwn-simulator/contracts";

/** Properties used by the virtualized event list. */
export interface VirtualEventListProps {
  /** Events rendered by the list in ascending sequence order. */
  events: Array<SimulationEvent>;
  /** Message displayed when no events are available. */
  emptyMessage: string;
  /** Keeps the viewport at the newest event while the user is near the end. */
  autoScroll: boolean;
  /** Loads older events when the viewport reaches the beginning. */
  onReachStart?: () => void;
  /** Notifies the parent whether the viewport is close to the newest event. */
  onNearEnd?: (nearEnd: boolean) => void;
}
