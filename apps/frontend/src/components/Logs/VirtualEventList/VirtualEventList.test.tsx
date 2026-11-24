import { render, screen } from "@testing-library/react";
import { RadioProtocol, SimulationEvent, SimulationEventType } from "@lwn-simulator/contracts";
import { describe, expect, it } from "vitest";
import { VirtualEventList } from "./VirtualEventList";

describe("payload evidence in simulation logs", () => {
  const event: SimulationEvent = {
    id: "event-1", sequence: 1, timestampMilliseconds: 1000,
    type: SimulationEventType.GatewayPacketReceived, message: "Frame received",
  };

  it("shows the actual decoded values", () => {
    render(<VirtualEventList events={[{ ...event, decodedPayload: { temperature: 21, humidity: 50 } }]} emptyMessage="No events" autoScroll={false} />);
    expect(screen.getByText(/Decoded.*temperature.*21.*humidity.*50/)).toBeInTheDocument();
  });

  it("shows codec errors instead of silently hiding failed decoding", () => {
    render(<VirtualEventList events={[{ ...event, error: "decodeUplink failed: bad payload" }]} emptyMessage="No events" autoScroll={false} />);
    expect(screen.getByText(/Error: decodeUplink failed: bad payload/)).toBeInTheDocument();
  });

  it("shows binary raw bytes without treating them as decoded application data", () => {
    render(<VirtualEventList events={[{ ...event, radioFrame: {
      protocol: RadioProtocol.RawLora, payload: "AN6tvu8=", senderID: "gateway",
      frequency: 868100000, bandwidth: 125000, spreadingFactor: 7, codingRate: "4/5",
      transmissionPower: 14, timestampMilliseconds: 1000,
    } }]} emptyMessage="No events" autoScroll={false} />);
    expect(screen.getByText(/Raw LoRa · HEX 00 DE AD BE EF/)).toBeInTheDocument();
    expect(screen.queryByText(/Decoded/)).not.toBeInTheDocument();
  });
});
