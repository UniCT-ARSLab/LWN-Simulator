import {
  Device,
  Gateway,
  SimulationEvent,
} from "@lwn-simulator/contracts";
import { SensorMapLink, SensorMapLogic } from "@/components/SensorMap";

/** Properties required to render the live map and event stream. */
export interface SimulationDashboardTopologyProps {
  /** Devices currently configured in the active profile. */
  devices: Array<Device>;
  /** Gateways currently configured in the active profile. */
  gateways: Array<Gateway>;
  /** Map controller returned by the sensor-map hook. */
  mapLogic: SensorMapLogic;
  /** Links currently active in the simulation. */
  links: Array<SensorMapLink>;
  /** Events received from the simulation engine. */
  events: Array<SimulationEvent>;
}
