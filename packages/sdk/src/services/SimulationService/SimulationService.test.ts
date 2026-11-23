import type { SimulationEvent } from "@lwn-simulator/contracts";
import { describe, expect, it } from "vitest";
import { SimulationService } from "./SimulationService";

const event = (id: string, sequence: number): SimulationEvent =>
  ({ id, sequence, message: id } as SimulationEvent);

describe("SimulationService event collection", () => {
  it("merges by event id, orders by sequence, and keeps the newest limit", () => {
    const current = [event("one", 1), event("two", 2)];
    const incoming = [event("two", 3), event("three", 4)];

    expect(SimulationService.mergeEvents(current, incoming, 2)).toEqual([
      event("two", 3),
      event("three", 4),
    ]);
  });

  it("appends an event without duplicating an existing id", () => {
    expect(
      SimulationService.appendEvent([event("one", 1)], event("one", 2)),
    ).toEqual([event("one", 2)]);
  });
});


describe("simulation log export URLs", () => {
  it("keeps downloads scoped to the current profile and encodes run identifiers", () => {
    const service = new SimulationService("http://localhost:8080/api/profiles/profile-1");
    expect(service.getLogExportURL()).toBe("http://localhost:8080/api/profiles/profile-1/simulation/logs/export");
    expect(service.getLogExportURL("run/1")).toBe("http://localhost:8080/api/profiles/profile-1/simulation/logs/run%2F1/export");
    expect(service.getTelemetryExportURL()).toBe("http://localhost:8080/api/profiles/profile-1/simulation/logs/telemetry");
    expect(service.getTelemetryExportURL("run/1")).toBe("http://localhost:8080/api/profiles/profile-1/simulation/logs/run%2F1/telemetry");
  });
});
