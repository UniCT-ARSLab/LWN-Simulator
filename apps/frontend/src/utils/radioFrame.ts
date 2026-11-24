import { RadioProtocol, SimulationEvent } from "@lwn-simulator/contracts";

/** Formats raw radio bytes from the JSON Base64 representation for event lists. */
export const formatRawRadioPayload = (
  frame: SimulationEvent["radioFrame"],
): string | null => {
  if (!frame || frame.protocol !== RadioProtocol.RawLora) return null;
  try {
    const bytes = atob(frame.payload);
    return `Raw LoRa · HEX ${Array.from(bytes, (byte) =>
      byte.charCodeAt(0).toString(16).padStart(2, "0").toUpperCase(),
    ).join(" ")}`;
  } catch {
    return `Raw LoRa · Base64 ${frame.payload}`;
  }
};
