import { SimulationSnapshot } from "@lwn-simulator/contracts";

/** Properties required to render the simulation metrics cards. */
export interface SimulationDashboardMetricsProps {
  /** Current simulation snapshot, when one has been received. */
  snapshot: SimulationSnapshot | null;
  /** Recent packet success-rate values used by the sparkline. */
  recentRates: Array<number>;
}
