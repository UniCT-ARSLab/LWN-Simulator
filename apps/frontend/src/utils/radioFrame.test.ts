import { RadioProtocol, SimulationEvent } from "@lwn-simulator/contracts";
import { describe, expect, it } from "vitest";
import { formatRawRadioPayload } from "./radioFrame";

describe("raw radio payload display", () => {
  const frame: NonNullable<SimulationEvent["radioFrame"]> = {
    protocol: RadioProtocol.RawLora,
    payload: "AN6tvu8=",
    bandwidth: 125000,
    codingRate: "4/5",
    frequency: 868100000,
    senderID: "gateway",
    spreadingFactor: 7,
    timestampMilliseconds: 1000,
    transmissionPower: 14,
  };

  it("displays binary bytes including zero without interpreting them as text", () => {
    expect(formatRawRadioPayload(frame)).toBe("Raw LoRa · HEX 00 DE AD BE EF");
  });

  it("keeps LoRaWAN events on the existing display path", () => {
    expect(formatRawRadioPayload({ ...frame, protocol: RadioProtocol.Lorawan })).toBeNull();
    expect(formatRawRadioPayload(undefined)).toBeNull();
  });

  it("preserves the original representation when an event has invalid Base64", () => {
    expect(formatRawRadioPayload({ ...frame, payload: "???" })).toBe("Raw LoRa · Base64 ???");
  });
});
