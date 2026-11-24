"use client";

import { MapLayerMouseEvent } from "maplibre-gl";
import {
  useSensorMapProps,
  SensorMapLogic,
  SensorMapMode,
  SensorMapPos,
} from "./SensorMap.types";
import { useCallback, useEffect, useState } from "react";

export const useSensorMap = (props: useSensorMapProps): SensorMapLogic => {
  const mode = props.mode || SensorMapMode.Sensor;

  // States
  const [markerPos, setMarkerPos] = useState<SensorMapPos | null>(null);
  const [selectedPos, setSelectedPos] = useState<SensorMapPos | null>(null);
  const [antennaRange, setAntennaRange] = useState<number | null>(null);

  // Functions
  const handleClick = useCallback((e: MapLayerMouseEvent) => {
    const { lat, lng } = e.lngLat;

    setSelectedPos({
      lat,
      lng,
    });
  }, []);

  const updatePos = useCallback((lat: number, lng: number) => {
    setMarkerPos((previous) => {
      if (previous && previous.lat === lat && previous.lng === lng) {
        return previous;
      }

      return { lat, lng };
    });
  }, []);

  // Effects
  useEffect(() => {
    if (selectedPos) setMarkerPos(selectedPos);
  }, [selectedPos]);

  return {
    handleClick: mode == SensorMapMode.Coords ? handleClick : undefined,
    selectedPos,
    markerPos,
    updatePos,
    antennaRange,
    setAntennaRange,
  };
};
