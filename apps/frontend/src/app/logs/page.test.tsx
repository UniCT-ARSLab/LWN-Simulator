import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import LogsPage from "./page";

const mocks = vi.hoisted(() => ({
  getLogs: vi.fn(),
  listener: null as null | ((message: string) => void),
  subscribe: vi.fn((listener: (message: string) => void) => {
    mocks.listener = listener;
    return () => { mocks.listener = null; };
  }),
}));
vi.mock("@lwn-simulator/sdk", () => ({
  useProfileService: () => ({ activeProfileID: "profile-1" }),
  useSimulationService: () => ({ getLogs: mocks.getLogs }),
  useWebSocketService: () => ({ connectionState: "connected", subscribe: mocks.subscribe }),
}));
vi.mock("@/components/Logs", () => ({ LogExportButton: () => null }));

describe("log history profile isolation", () => {
  beforeEach(() => mocks.getLogs.mockResolvedValue({ runs: [], total: 0 }));

  it("ignores foreign profile run creations and updates", async () => {
    render(<LogsPage />);
    await screen.findByText("No simulation logs yet");
    for (const type of ["simulation.run.created", "simulation.run.updated"]) {
      act(() => mocks.listener?.(JSON.stringify({ type, profileID: "profile-2", runSummary: { id: "foreign" } })));
    }
    expect(screen.getByText("No simulation logs yet")).toBeInTheDocument();
  });

  it("receives current-profile run updates", async () => {
    render(<LogsPage />);
    await screen.findByText("No simulation logs yet");
    act(() => mocks.listener?.(JSON.stringify({ type: "simulation.run.created", profileID: "profile-1", runSummary: {
      id: "current", startedAtMilliseconds: 1000, status: "running", durationMilliseconds: 0, eventCount: 7, deviceCount: 1, gatewayCount: 1, packetSuccessRate: 0,
    } })));
    expect(screen.getByRole("link", { name: /Running/ })).toHaveAttribute("href", "/logs/detail?runId=current");
  });
});
