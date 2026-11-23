import { DeviceRegion } from "@lwn-simulator/contracts";
import { NotificationHandler } from "@lwn-simulator/ui-components";

export const rx1DataRateOffsetRange: Record<DeviceRegion, [number, number]> = {
  EU868: [0, 7],
  US915: [0, 3],
  CN779: [0, 5],
  EU433: [0, 7],
  AU915: [0, 3],
  CN470: [0, 7],
  AS923: [0, 7],
  KR920: [0, 5],
  IN865: [0, 7],
  RU864: [0, 7],
};

export const rx2DataRateOptions: Record<DeviceRegion, number[]> = {
  EU868: [0, 1, 2, 3, 4, 5, 6, 7],
  US915: [8, 9, 10, 11, 12, 13],
  CN779: [0, 1, 2, 3, 4, 5],
  EU433: [0, 1, 2, 3, 4, 5],
  AU915: [8, 9, 10, 11, 12, 13],
  CN470: [0, 1, 2, 3, 4, 5],
  AS923: [2, 3, 4, 5],
  KR920: [0, 1, 2, 3, 4, 5],
  IN865: [0, 1, 2, 3, 4, 5],
  RU864: [0, 1, 2, 3, 4, 5],
};

export const rx2Frequency: Record<DeviceRegion, number> = {
  EU868: 869525000,
  US915: 923300000,
  CN779: 786000000,
  EU433: 434665000,
  AU915: 923300000,
  CN470: 505300000,
  AS923: 923200000,
  KR920: 923300000,
  IN865: 866550000,
  RU864: 869100000,
};

export const lngValidation =
  /^-?(?:180(?:\.0+)?|1[0-7]\d(?:\.\d+)?|[0-9]?\d(?:\.\d+)?)$/;

export const latValidation = /^-?(?:90(?:\.0+)?|(?:[0-8]?\d)(?:\.\d+)?)$/;

export const estimateRegion = (
  lat: number,
  lng: number,
): DeviceRegion | null => {
  // Europe
  if (lat >= 35 && lat <= 72 && lng >= -25 && lng <= 45) {
    return DeviceRegion.EU868;
  }

  // Russia
  if (lat >= 41 && lat <= 82 && lng >= 19 && lng <= 180) {
    return DeviceRegion.RU864;
  }

  // South Korea
  if (lat >= 33 && lat <= 39 && lng >= 124 && lng <= 132) {
    return DeviceRegion.KR920;
  }

  // India
  if (lat >= 6 && lat <= 37 && lng >= 68 && lng <= 98) {
    return DeviceRegion.IN865;
  }

  // China
  if (lat >= 18 && lat <= 54 && lng >= 73 && lng <= 135) {
    return DeviceRegion.CN470;
  }

  // Australia
  if (lat >= -44 && lat <= -10 && lng >= 112 && lng <= 154) {
    return DeviceRegion.AU915;
  }

  // USA / Canada / part of Americas
  if (lat >= 24 && lat <= 72 && lng >= -170 && lng <= -50) {
    return DeviceRegion.US915;
  }

  // South America / Japan / Taiwan / SE Asia etc.
  if (lat >= -50 && lat <= 50 && lng >= 95 && lng <= 180) {
    return DeviceRegion.AS923;
  }

  return null;
};

export const randomDevEUI = (): string => {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);

  bytes[0] = (bytes[0] & 0b11111100) | 0b00000010;

  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join(
    "",
  );
};

export const randomGatewayEUI = (): string => {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);

  bytes[0] = (bytes[0] & 0b11111100) | 0b00000010;

  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0"))
    .join("")
    .toUpperCase();
};

export const getAltitude = async (
  lat: number,
  lng: number,
): Promise<number> => {
  let alt = 0;

  try {
    const response = await fetch(
      `https://api.open-elevation.com/api/v1/lookup?locations=${lat},${lng}`,
    );

    if (!response.ok) {
      throw new Error("Altitude service returned an error");
    }

    const data = await response.json();

    alt = data.results[0].elevation as number;
  } catch {
    NotificationHandler.instance.error("Failed to retrieve altitude");
  }

  return alt;
};
