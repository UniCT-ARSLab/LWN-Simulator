import { MapLayerMouseEvent } from "maplibre-gl";
import { Device, Gateway } from "@lwn-simulator/contracts";

/** Interaction modes supported by the sensor map. */
export enum SensorMapMode {
  /** Shows configured devices and gateways. */
  Sensor = "sensor",
  /** Allows selecting geographic coordinates. */
  Coords = "coords",
}

/** Geographic coordinate expressed in latitude and longitude. */
export interface Marker {
  /** Latitude in decimal degrees. */
  latitude: number;
  /** Longitude in decimal degrees. */
  longitude: number;
}

/** Geographic position used by map interaction state. */
export interface SensorMapPos {
  /** Latitude in decimal degrees. */
  lat: number;
  /** Longitude in decimal degrees. */
  lng: number;
}

/** Options accepted by the sensor-map state hook. */
export interface useSensorMapProps {
  /** Map interaction mode. */
  mode?: SensorMapMode;
}

/** State and actions exposed by the sensor-map state hook. */
export interface SensorMapLogic {
  /** Handles map layer clicks when coordinate selection is enabled. */
  handleClick?: (e: MapLayerMouseEvent) => void;
  /** Last selected coordinate. */
  selectedPos: SensorMapPos | null;
  /** Current marker coordinate. */
  markerPos: SensorMapPos | null;
  /** Updates the marker coordinate. */
  updatePos: (lat: number, lng: number) => void;
  /** Device antenna range currently visualized on the map. */
  antennaRange: number | null;
  /** Updates the displayed antenna range. */
  setAntennaRange: (val: number | null) => void;
}

/** Entity kind and value used when building map markers. */
export type SensorMapEntity =
  | { kind: "device"; entity: Device }
  | { kind: "gateway"; entity: Gateway };

/** Line connecting a communicating device and gateway. */
export interface SensorMapLink {
  /** Stable identifier for the link. */
  id: string;
  /** Device endpoint coordinate. */
  from: Marker;
  /** Gateway endpoint coordinate. */
  to: Marker;
}

/** Properties required to render the sensor map. */
export interface SensorMapProps {
  /** State and actions controlling the map. */
  logic: SensorMapLogic;
  /** Map interaction mode. */
  mode?: SensorMapMode;
  /** Devices displayed on the map. */
  devices?: Array<Device>;
  /** Gateways displayed on the map. */
  gateways?: Array<Gateway>;
  /** Active communication links displayed on the map. */
  links?: Array<SensorMapLink>;
}
