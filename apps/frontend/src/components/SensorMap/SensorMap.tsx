"use client";

import {
  SensorMapMode,
  SensorMapProps,
  SensorMapEntity,
} from "./SensorMap.types";
import MapGL, {
  Layer,
  MapRef,
  Marker,
  Popup,
  Source,
} from "react-map-gl/maplibre";
import "maplibre-gl/dist/maplibre-gl.css";
import {
  Theme,
  useOutsideAlerter,
  useThemeService,
} from "@lwn-simulator/ui-components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { MapLayerMouseEvent } from "maplibre-gl";

const DEFAULT_POSITION = { longitude: 15.083, latitude: 37.5079, zoom: 12 };
const DENSE_DEVICE_LAYER_ID = "sensor-map-dense-devices";
const DENSE_GATEWAY_LAYER_ID = "sensor-map-dense-gateways";
const DEVICE_ICON_ID = "sensor-map-device-icon";
const GATEWAY_ICON_ID = "sensor-map-gateway-icon";
const MATERIAL_ICON_SIZE = 64;

const formatEnumValue = (value: string): string => value.replace(/[-_]/g, " ");

const SensorMap = (props: SensorMapProps) => {
  const [selectedEntity, setSelectedEntity] = useState<SensorMapEntity | null>(
    null,
  );
  const [materialIconsReady, setMaterialIconsReady] = useState(false);
  const [webglAvailable, setWebglAvailable] = useState<boolean | null>(null);
  const themeService = useThemeService();
  const mapRef = useRef<MapRef | null>(null);
  const tooltipRef = useRef<HTMLDivElement | null>(null);
  const linksFittedRef = useRef(false);
  const baseColor = themeService.theme === Theme.Dark ? "#000000" : "#ffffff";

  useEffect(() => {
    const canvas = document.createElement("canvas");
    const handleContextCreationError = (event: Event) => {
      event.preventDefault();
    };

    canvas.addEventListener(
      "webglcontextcreationerror",
      handleContextCreationError,
    );

    try {
      setWebglAvailable(
        Boolean(
          canvas.getContext("webgl2", {
            failIfMajorPerformanceCaveat: false,
            powerPreference: "high-performance",
          }),
        ),
      );
    } catch {
      setWebglAvailable(false);
    } finally {
      canvas.removeEventListener(
        "webglcontextcreationerror",
        handleContextCreationError,
      );
    }
  }, []);

  const registerMaterialIcons = useCallback(async () => {
    const map = mapRef.current?.getMap();
    if (!map) return;

    await document.fonts.load('700 48px "Material Symbols Outlined"');

    for (const [id, ligature] of [
      [DEVICE_ICON_ID, "sensors"],
      [GATEWAY_ICON_ID, "router"],
    ] as const) {
      if (map.hasImage(id)) continue;

      const canvas = document.createElement("canvas");
      canvas.width = MATERIAL_ICON_SIZE;
      canvas.height = MATERIAL_ICON_SIZE;
      const context = canvas.getContext("2d");
      if (!context) continue;

      context.font = '700 48px "Material Symbols Outlined"';
      context.textAlign = "center";
      context.textBaseline = "middle";
      context.fillStyle = "#ffffff";
      context.fillText(
        ligature,
        MATERIAL_ICON_SIZE / 2,
        MATERIAL_ICON_SIZE / 2,
      );

      map.addImage(
        id,
        {
          width: MATERIAL_ICON_SIZE,
          height: MATERIAL_ICON_SIZE,
          data: context.getImageData(
            0,
            0,
            MATERIAL_ICON_SIZE,
            MATERIAL_ICON_SIZE,
          ).data,
        },
        { pixelRatio: 2, sdf: true },
      );
    }

    setMaterialIconsReady(true);
  }, []);

  useOutsideAlerter({
    ref: tooltipRef,
    onClickOutside: () => setSelectedEntity(null),
  });

  const createCircle = (
    longitude: number,
    latitude: number,
    radiusMeters: number,
    points = 64,
  ) => {
    const coordinates: [number, number][] = [];
    const earthRadius = 6371008.8;
    const lat = (latitude * Math.PI) / 180;
    const lon = (longitude * Math.PI) / 180;
    const angularDistance = radiusMeters / earthRadius;

    for (let i = 0; i <= points; i += 1) {
      const bearing = (i / points) * 2 * Math.PI;
      const circleLat = Math.asin(
        Math.sin(lat) * Math.cos(angularDistance) +
          Math.cos(lat) * Math.sin(angularDistance) * Math.cos(bearing),
      );
      const circleLon =
        lon +
        Math.atan2(
          Math.sin(bearing) * Math.sin(angularDistance) * Math.cos(lat),
          Math.cos(angularDistance) - Math.sin(lat) * Math.sin(circleLat),
        );
      coordinates.push([
        (circleLon * 180) / Math.PI,
        (circleLat * 180) / Math.PI,
      ]);
    }

    return {
      type: "Feature" as const,
      geometry: { type: "Polygon" as const, coordinates: [coordinates] },
      properties: {},
    };
  };

  const centerOnMarker = useCallback(() => {
    const marker = props.logic.markerPos;
    if (props.mode !== SensorMapMode.Coords || !marker || !mapRef.current)
      return;
    mapRef.current.flyTo({
      center: [marker.lng, marker.lat],
      zoom: 15,
      duration: 0,
    });
  }, [props.logic.markerPos, props.mode]);

  const fitCommunicationLinks = useCallback(() => {
    if (
      props.mode === SensorMapMode.Coords ||
      linksFittedRef.current ||
      !mapRef.current ||
      !props.links?.length
    )
      return;

    const coordinates = props.links.flatMap((link) => [link.from, link.to]);
    const longitudes = coordinates.map((coordinate) => coordinate.longitude);
    const latitudes = coordinates.map((coordinate) => coordinate.latitude);

    mapRef.current.fitBounds(
      [
        [Math.min(...longitudes), Math.min(...latitudes)],
        [Math.max(...longitudes), Math.max(...latitudes)],
      ],
      { padding: 72, maxZoom: 14, duration: 0 },
    );
    linksFittedRef.current = true;
  }, [props.links, props.mode]);

  useEffect(() => {
    centerOnMarker();
  }, [centerOnMarker]);
  useEffect(() => {
    fitCommunicationLinks();
  }, [fitCommunicationLinks, props.links?.length]);

  const selectedPosition = selectedEntity
    ? selectedEntity.kind === "device"
      ? {
          latitude: selectedEntity.entity.locationConfig.latitude,
          longitude: selectedEntity.entity.locationConfig.longitude,
        }
      : {
          latitude: selectedEntity.entity.latitude,
          longitude: selectedEntity.entity.longitude,
        }
    : null;

  const devicesByID = useMemo(
    () => new Map(props.devices?.map((device) => [device.id, device]) ?? []),
    [props.devices],
  );
  const gatewaysByID = useMemo(
    () =>
      new Map(props.gateways?.map((gateway) => [gateway.id, gateway]) ?? []),
    [props.gateways],
  );
  const denseEntities = useMemo(
    () => ({
      type: "FeatureCollection" as const,
      features: [
        ...(props.devices ?? []).map((device) => ({
          type: "Feature" as const,
          geometry: {
            type: "Point" as const,
            coordinates: [
              device.locationConfig.longitude,
              device.locationConfig.latitude,
            ],
          },
          properties: { id: device.id, kind: "device" },
        })),
        ...(props.gateways ?? []).map((gateway) => ({
          type: "Feature" as const,
          geometry: {
            type: "Point" as const,
            coordinates: [gateway.longitude, gateway.latitude],
          },
          properties: { id: gateway.id, kind: "gateway" },
        })),
      ],
    }),
    [props.devices, props.gateways],
  );
  const linkLines = useMemo(
    () => ({
      type: "FeatureCollection" as const,
      features: (props.links ?? []).map((link) => ({
        type: "Feature" as const,
        geometry: {
          type: "LineString" as const,
          coordinates: [
            [link.from.longitude, link.from.latitude],
            [link.to.longitude, link.to.latitude],
          ],
        },
        properties: { id: link.id },
      })),
    }),
    [props.links],
  );
  const handleMapClick = useCallback(
    (event: MapLayerMouseEvent) => {
      const feature = event.features?.[0];
      const id = String(feature?.properties?.id ?? "");

      if (feature?.layer.id === DENSE_DEVICE_LAYER_ID) {
        const device = devicesByID.get(id);
        if (device) {
          setSelectedEntity({ kind: "device", entity: device });
          return;
        }
      }
      if (feature?.layer.id === DENSE_GATEWAY_LAYER_ID) {
        const gateway = gatewaysByID.get(id);
        if (gateway) {
          setSelectedEntity({ kind: "gateway", entity: gateway });
          return;
        }
      }

      props.logic.handleClick?.(event);
    },
    [devicesByID, gatewaysByID, props.logic],
  );

  return (
    <div className="sensor-map">
      {webglAvailable === null && (
        <div className="sensor-map-status" role="status">
          <strong>Loading map</strong>
        </div>
      )}
      {webglAvailable === false && (
        <div className="sensor-map-status" role="alert">
          <strong>Map unavailable</strong>
          <p>
            WebGL2 could not be initialized. Enable hardware acceleration and
            restart the browser or desktop application. If the problem
            persists, check that the app was not started with GPU disabled.
          </p>
        </div>
      )}
      {webglAvailable && (
      <MapGL
        ref={mapRef}
        onLoad={() => {
          void registerMaterialIcons();
          centerOnMarker();
          fitCommunicationLinks();
        }}
        onClick={handleMapClick}
        interactiveLayerIds={[DENSE_DEVICE_LAYER_ID, DENSE_GATEWAY_LAYER_ID]}
        initialViewState={DEFAULT_POSITION}
        style={{ width: "100%", height: "100%" }}
        mapStyle={
          themeService.theme === Theme.Dark
            ? "/dark-map-theme.json"
            : "/light-map-theme.json"
        }
        attributionControl={false}
        dragRotate={false}
        touchZoomRotate={false}
      >
        {linkLines.features.length > 0 && (
          <Source id="sensor-map-links" type="geojson" data={linkLines}>
            <Layer
              id="sensor-map-links-glow"
              type="line"
              paint={{
                "line-color":
                  themeService.theme === Theme.Dark ? "#38bdf8" : "#005cff",
                "line-width": 8,
                "line-opacity": 0.3,
                "line-blur": 2,
              }}
            />
            <Layer
              id="sensor-map-links-line"
              type="line"
              paint={{
                "line-color":
                  themeService.theme === Theme.Dark ? "#38bdf8" : "#005cff",
                "line-width": 4,
                "line-opacity": 1,
                "line-dasharray": [2, 1.5],
              }}
            />
          </Source>
        )}

        {materialIconsReady ? (
          <Source
            id="sensor-map-dense-entities"
            type="geojson"
            data={denseEntities}
          >
            <Layer
              id={DENSE_DEVICE_LAYER_ID}
              type="symbol"
              filter={["==", ["get", "kind"], "device"]}
              layout={{
                "icon-image": DEVICE_ICON_ID,
                "icon-size": 0.75,
                "icon-allow-overlap": true,
              }}
              paint={{
                "icon-color":
                  themeService.theme === Theme.Dark ? "#38bdf8" : "#005cff",
                "icon-halo-color": baseColor,
                "icon-halo-width": 3,
              }}
            />
            <Layer
              id={DENSE_GATEWAY_LAYER_ID}
              type="symbol"
              filter={["==", ["get", "kind"], "gateway"]}
              layout={{
                "icon-image": GATEWAY_ICON_ID,
                "icon-size": 0.75,
                "icon-allow-overlap": true,
              }}
              paint={{
                "icon-color":
                  themeService.theme === Theme.Dark ? "#f59e0b" : "#c53903",
                "icon-halo-color": baseColor,
                "icon-halo-width": 3,
              }}
            />
          </Source>
        ) : null}

        {selectedEntity && selectedPosition && (
          <Popup
            longitude={selectedPosition.longitude}
            latitude={selectedPosition.latitude}
            anchor="bottom"
            maxWidth="min(21rem, calc(100vw - 2rem))"
            closeButton={false}
            closeOnClick={false}
            onClose={() => setSelectedEntity(null)}
            className="sensor-map-tooltip"
          >
            <div ref={tooltipRef} className="sensor-map-tooltip-content">
              <div className="sensor-map-tooltip-header">
                <span className="sensor-map-tooltip-kind">
                  {selectedEntity.kind === "device" ? "Sensors" : "Router"}
                </span>
                <div>
                  <strong>{selectedEntity.entity.name}</strong>
                  <span>
                    {selectedEntity.kind === "device" ? "Device" : "Gateway"}
                  </span>
                </div>
                <button
                  type="button"
                  aria-label="Close tooltip"
                  onClick={() => setSelectedEntity(null)}
                >
                  ×
                </button>
              </div>
              <div className="sensor-map-tooltip-details">
                <span>
                  Status{" "}
                  <strong
                    className={
                      selectedEntity.entity.active ? "is-active" : "is-inactive"
                    }
                  >
                    {selectedEntity.entity.active ? "Active" : "Inactive"}
                  </strong>
                </span>
                {selectedEntity.kind === "device" ? (
                  <>
                    <span>
                      DevEUI <strong>{selectedEntity.entity.devEUI}</strong>
                    </span>
                    <span>
                      Class{" "}
                      <strong>
                        {formatEnumValue(selectedEntity.entity.class)}
                      </strong>
                    </span>
                    <span>
                      Region{" "}
                      <strong>
                        {selectedEntity.entity.locationConfig.region}
                      </strong>
                    </span>
                  </>
                ) : (
                  <>
                    <span>
                      Gateway EUI{" "}
                      <strong>{selectedEntity.entity.gatewayEUI}</strong>
                    </span>
                    <span>
                      Type{" "}
                      <strong>
                        {formatEnumValue(selectedEntity.entity.type)}
                      </strong>
                    </span>
                  </>
                )}
              </div>
            </div>
          </Popup>
        )}

        {selectedEntity?.kind === "device" && (
          <Source
            id="device-coverage-circle"
            type="geojson"
            data={createCircle(
              selectedPosition!.longitude,
              selectedPosition!.latitude,
              selectedEntity.entity.advancedConfig.antennaRange,
            )}
          >
            <Layer
              id="device-coverage-circle-line"
              type="line"
              paint={{
                "line-color": "#fff",
                "line-width": 2,
                "line-opacity": 0.9,
                "line-dasharray": [2, 4],
              }}
            />
          </Source>
        )}

        {props.logic.markerPos && (
          <>
            <Marker
              longitude={props.logic.markerPos.lng}
              latitude={props.logic.markerPos.lat}
              anchor="bottom"
            >
              <span className="material-symbols-outlined pos-marker">
                location_on
              </span>
            </Marker>
            {props.logic.antennaRange && (
              <Source
                id="coverage-circle"
                type="geojson"
                data={createCircle(
                  props.logic.markerPos.lng,
                  props.logic.markerPos.lat,
                  props.logic.antennaRange,
                )}
              >
                <Layer
                  id="coverage-circle-line"
                  type="line"
                  paint={{
                    "line-color": "#fff",
                    "line-width": 2,
                    "line-opacity": 0.8,
                    "line-dasharray": [2, 4],
                  }}
                />
              </Source>
            )}
          </>
        )}
      </MapGL>
      )}
    </div>
  );
};

export default SensorMap;
