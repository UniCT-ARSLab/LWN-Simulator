import template from "./map-theme-template.json";
import path from "path";
import fs from "fs";

interface Theme {
  water: string;
  land: string;
  buildings: string;
  roads: string;
  text: string;
  halo: string;
  primary: string;
  secondary: string;
  boundaries: string;
}

const darkMapTheme: Theme = {
  water: "#031427",
  land: "#0b1c30",
  buildings: "#1b2b3f",
  roads: "#3b536b",
  text: "#c6c6cd",
  halo: "#031427",
  primary: "#38bdf8",
  secondary: "#f59e0b",
  boundaries: "#45464d",
};

const lightMapTheme: Theme = {
  water: "#a8d5f2",
  land: "#e5e3df",
  buildings: "#fdfdfd",
  roads: "#b0b4bb",
  text: "#202124",
  halo: "#ffffff",
  primary: "#1a73e8",
  secondary: "#ea4335",
  boundaries: "#9aa0a6",
};

const setColor = (
  res: typeof template,
  id: string,
  prop: Record<string, unknown>,
) => {
  const layer = res.layers.find((l) => l.id == id);

  if (layer) {
    const [key] = Object.keys(prop);
    layer[key as keyof typeof layer] = Object.values(prop)[0] as never;
  }
};

const genTheme = (theme: Theme): typeof template => {
  const res = { ...template };

  setColor(res, "water", {
    paint: {
      "fill-color": theme.water,
    },
  });

  setColor(res, "background", {
    paint: {
      "background-color": theme.land,
    },
  });

  setColor(res, "landuse_residential", {
    paint: {
      "fill-color": {
        base: 1,
        stops: [
          [9, `${theme.land}D6`],
          [12, `${theme.land}7D`],
        ],
      },
    },
  });

  setColor(res, "landuse_pitch", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "landuse_track", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "landuse_cemetery", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "landuse_hospital", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "landuse_school", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "aeroway_fill", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "aeroway_taxiway", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [11, 0.5],
          [20, 6],
        ],
      },
    },
  });

  setColor(res, "aeroway_runway", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [11, 3],
          [20, 16],
        ],
      },
    },
  });

  setColor(res, "landcover_ice", {
    paint: {
      "fill-antialias": false,
      "fill-color": theme.land,
      "fill-opacity": 0.8,
    },
  });

  setColor(res, "landcover_sand", {
    paint: { "fill-color": theme.land },
  });

  setColor(res, "landcover_wetland", {
    paint: {
      "fill-antialias": true,
      "fill-opacity": 0.8,
      "fill-color": theme.land,
    },
  });

  setColor(res, "waterway_tunnel", {
    paint: {
      "line-color": theme.primary,
      "line-dasharray": [3, 3],
      "line-gap-width": {
        stops: [
          [12, 0],
          [20, 6],
        ],
      },
      "line-opacity": 1,
      "line-width": {
        base: 1.4,
        stops: [
          [8, 1],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "waterway_river", {
    paint: {
      "line-color": theme.primary,
      "line-width": {
        base: 1.2,
        stops: [
          [11, 0.5],
          [20, 6],
        ],
      },
    },
  });

  setColor(res, "waterway_other", {
    paint: {
      "line-color": theme.primary,
      "line-width": {
        base: 1.3,
        stops: [
          [13, 0.5],
          [20, 6],
        ],
      },
    },
  });

  setColor(res, "park", {
    paint: {
      "fill-color": theme.land,
      "fill-opacity": 0.7,
      "fill-outline-color": theme.land,
    },
  });

  setColor(res, "park_outline", {
    paint: {
      "line-dasharray": [1, 1.5],
      "line-color": theme.land,
    },
  });

  setColor(res, "landcover_wood", {
    paint: {
      "fill-antialias": false,
      "fill-color": theme.land,
      "fill-opacity": 0.4,
    },
  });

  setColor(res, "landcover_grass", {
    paint: {
      "fill-antialias": false,
      "fill-color": theme.land,
      "fill-opacity": 0.3,
    },
  });

  setColor(res, "building", {
    paint: {
      "fill-color": theme.buildings,
      "fill-outline-color": {
        base: 1,
        stops: [
          [13, `${theme.buildings}52`],
          [14, `${theme.buildings}99`],
        ],
      },
    },
  });

  setColor(res, "building-3d", {
    paint: {
      "fill-extrusion-color": theme.buildings,
      "fill-extrusion-height": {
        property: "render_height",
        type: "identity",
      },
      "fill-extrusion-base": {
        property: "render_min_height",
        type: "identity",
      },
      "fill-extrusion-opacity": 0.8,
    },
  });

  setColor(res, "road_motorway", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "road_trunk_primary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "road_secondary_tertiary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [6.5, 0],
          [8, 0.5],
          [20, 13],
        ],
      },
    },
  });

  setColor(res, "road_minor", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [13.5, 0],
          [14, 2.5],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "road_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "tunnel_motorway_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.5, 0.25],
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "tunnel_service_track_casing", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.5, 0.25],
      "line-width": {
        base: 1.2,
        stops: [
          [15, 1],
          [16, 4],
          [20, 11],
        ],
      },
    },
  });

  setColor(res, "tunnel_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "tunnel_street_casing", {
    paint: {
      "line-color": theme.roads,
      "line-opacity": {
        stops: [
          [12, 0],
          [12.5, 1],
        ],
      },
      "line-width": {
        base: 1.2,
        stops: [
          [12, 0.5],
          [13, 1],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "tunnel_secondary_tertiary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [8, 1.5],
          [20, 17],
        ],
      },
    },
  });

  setColor(res, "tunnel_trunk_primary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "tunnel_motorway_casing", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.5, 0.25],
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "road_motorway_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "road_service_track_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [15, 1],
          [16, 4],
          [20, 11],
        ],
      },
    },
  });

  setColor(res, "road_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "road_minor_casing", {
    paint: {
      "line-color": theme.roads,
      "line-opacity": {
        stops: [
          [12, 0],
          [12.5, 1],
        ],
      },
      "line-width": {
        base: 1.2,
        stops: [
          [12, 0.5],
          [13, 1],
          [14, 4],
          [20, 20],
        ],
      },
    },
  });

  setColor(res, "road_secondary_tertiary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [8, 1.5],
          [20, 17],
        ],
      },
    },
  });

  setColor(res, "road_trunk_primary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "road_motorway_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "bridge_motorway_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "bridge_service_track_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [15, 1],
          [16, 4],
          [20, 11],
        ],
      },
    },
  });

  setColor(res, "bridge_link_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12, 1],
          [13, 3],
          [14, 4],
          [20, 15],
        ],
      },
    },
  });

  setColor(res, "bridge_street_casing", {
    paint: {
      "line-color": theme.roads,
      "line-opacity": {
        stops: [
          [12, 0],
          [12.5, 1],
        ],
      },
      "line-width": {
        base: 1.2,
        stops: [
          [12, 0.5],
          [13, 1],
          [14, 4],
          [20, 25],
        ],
      },
    },
  });

  setColor(res, "bridge_path_pedestrian_casing", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [1, 0],
      "line-width": {
        base: 1.2,
        stops: [
          [14, 1.5],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "bridge_secondary_tertiary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [8, 1.5],
          [20, 17],
        ],
      },
    },
  });

  setColor(res, "bridge_trunk_primary_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "bridge_motorway_casing", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0.4],
          [6, 0.7],
          [7, 1.75],
          [20, 22],
        ],
      },
    },
  });

  setColor(res, "bridge_path_pedestrian", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [1, 0.3],
      "line-width": {
        base: 1.2,
        stops: [
          [14, 0.5],
          [20, 10],
        ],
      },
    },
  });

  setColor(res, "bridge_motorway_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "bridge_service_track", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [15.5, 0],
          [16, 2],
          [20, 7.5],
        ],
      },
    },
  });

  setColor(res, "bridge_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "bridge_street", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [13.5, 0],
          [14, 2.5],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "bridge_secondary_tertiary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [6.5, 0],
          [7, 0.5],
          [20, 10],
        ],
      },
    },
  });

  setColor(res, "bridge_trunk_primary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0],
          [7, 1],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "bridge_motorway", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0],
          [7, 1],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "bridge_major_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "bridge_major_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "bridge_transit_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "bridge_transit_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "tunnel_path_pedestrian", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [1, 0.75],
      "line-width": {
        base: 1.2,
        stops: [
          [14, 0.5],
          [20, 10],
        ],
      },
    },
  });

  setColor(res, "tunnel_motorway_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "tunnel_service_track", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [15.5, 0],
          [16, 2],
          [20, 7.5],
        ],
      },
    },
  });

  setColor(res, "tunnel_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "tunnel_minor", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [13.5, 0],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "tunnel_secondary_tertiary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [6.5, 0],
          [7, 0.5],
          [20, 10],
        ],
      },
    },
  });

  setColor(res, "tunnel_trunk_primary", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0],
          [7, 1],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "tunnel_motorway", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [5, 0],
          [7, 1],
          [20, 18],
        ],
      },
    },
  });

  setColor(res, "tunnel_major_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "tunnel_major_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "tunnel_transit_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "tunnel_transit_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "road_path_pedestrian", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [1, 0.7],
      "line-width": {
        base: 1.2,
        stops: [
          [14, 1],
          [20, 10],
        ],
      },
    },
  });

  setColor(res, "road_motorway_link", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [12.5, 0],
          [13, 1.5],
          [14, 2.5],
          [20, 11.5],
        ],
      },
    },
  });

  setColor(res, "road_service_track", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.2,
        stops: [
          [15.5, 0],
          [16, 2],
          [20, 7.5],
        ],
      },
    },
  });

  setColor(res, "road_major_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "road_major_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "road_transit_rail", {
    paint: {
      "line-color": theme.roads,
      "line-width": {
        base: 1.4,
        stops: [
          [14, 0.4],
          [15, 0.75],
          [20, 2],
        ],
      },
    },
  });

  setColor(res, "road_transit_rail_hatching", {
    paint: {
      "line-color": theme.roads,
      "line-dasharray": [0.2, 8],
      "line-width": {
        base: 1.4,
        stops: [
          [14.5, 0],
          [15, 3],
          [20, 8],
        ],
      },
    },
  });

  setColor(res, "boundary_3", {
    paint: {
      "line-color": theme.boundaries,
      "line-dasharray": [5, 1],
      "line-width": {
        base: 1,
        stops: [
          [4, 0.4],
          [5, 1],
          [12, 1.8],
        ],
      },
    },
  });

  setColor(res, "boundary_2_z0-4", {
    paint: {
      "line-color": "transparent",
      "line-opacity": {
        base: 1,
        stops: [
          [0, 0.4],
          [4, 1],
        ],
      },
      "line-width": {
        base: 1,
        stops: [
          [3, 1],
          [5, 1.2],
          [12, 3],
        ],
      },
    },
  });

  setColor(res, "boundary_2_z5-", {
    paint: {
      "line-color": "transparent",
      "line-opacity": {
        base: 1,
        stops: [
          [0, 0.4],
          [4, 1],
        ],
      },
      "line-width": {
        base: 1,
        stops: [
          [3, 1],
          [5, 1.2],
          [12, 3],
        ],
      },
    },
  });

  setColor(res, "water_name_line", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "water_name_point", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_z16", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 0.5,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_z16", {
    layout: {
      "text-anchor": "top",
      "text-field": "{name}",
      "text-font": ["Roboto Condensed Italic"],
      "text-max-width": 9,
      "text-offset": [0, 0.6],
      "text-size": 12,
    },
  });

  setColor(res, "poi_z15", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 0.5,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_z15", {
    layout: {
      "text-anchor": "top",
      "text-field": "{name}",
      "text-font": ["Roboto Condensed Italic"],
      "text-max-width": 9,
      "text-offset": [0, 0.6],
      "text-size": 12,
    },
  });

  setColor(res, "poi_z14", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 0.5,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_z14", {
    layout: {
      "text-anchor": "top",
      "text-field": "{name}",
      "text-font": ["Roboto Condensed Italic"],
      "text-max-width": 9,
      "text-offset": [0, 0.6],
      "text-size": 12,
    },
  });

  setColor(res, "road_label", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 0.5,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "continent", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "place_other", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1.2,
    },
  });

  setColor(res, "place_village", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1.2,
    },
  });

  setColor(res, "place_town", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1.2,
    },
  });

  setColor(res, "place_city", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1.2,
    },
  });

  setColor(res, "country_3", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 1,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "country_2", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 1,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "country_1", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 1,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "state", {
    paint: {
      "text-color": theme.text,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_transit", {
    paint: {
      "text-color": theme.text,
      "text-halo-blur": 0.5,
      "text-halo-color": theme.halo,
      "text-halo-width": 1,
    },
  });

  setColor(res, "poi_transit", {
    layout: {
      "text-anchor": "left",
      "text-field": "{name_en}",
      "text-font": ["Roboto Condensed Italic"],
      "text-max-width": 9,
      "text-offset": [0.9, 0],
      "text-size": 12,
    },
  });

  return res;
};

const main = () => {
  fs.writeFileSync(
    path.join(__dirname, `../../public/light-map-theme.json`),
    JSON.stringify(genTheme(lightMapTheme)),
  );
  fs.writeFileSync(
    path.join(__dirname, `../../public/dark-map-theme.json`),
    JSON.stringify(genTheme(darkMapTheme)),
  );
};

main();
