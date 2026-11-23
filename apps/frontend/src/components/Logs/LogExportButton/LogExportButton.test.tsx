import { act, fireEvent, render, screen } from "@testing-library/react";
import { NotificationHandler } from "@lwn-simulator/ui-components";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LogExportButton } from "./LogExportButton";

vi.mock("@lwn-simulator/sdk", () => ({
  useSimulationService: () => ({
    getLogExportURL: (id?: string) => id ? `/api/logs/${id}/export` : "/api/logs/export",
    getTelemetryExportURL: (id?: string) => id ? `/api/logs/${id}/telemetry` : "/api/logs/telemetry",
  }),
}));

describe("simulation log downloads", () => {
  const createObjectURL = vi.fn(() => "blob:log-export");
  const revokeObjectURL = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL });
  });

  afterEach(() => {
    act(() => vi.runOnlyPendingTimers());
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it.each([
    ["run-1", "Export JSON", "simulation-run-1.json", false],
    [undefined, "Export all (ZIP)", "simulation-logs.zip", false],
    ["run-1", "Export telemetry", "simulation-run-1-telemetry.zip", true],
    [undefined, "Export telemetry summary", "simulation-telemetry-summary.csv", true],
  ])("downloads %s with %s", async (runID, label, filename, telemetry) => {
    const blob = new Blob(["export"]);
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, blob: async () => blob });
    vi.stubGlobal("fetch", fetchMock);
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    render(<LogExportButton runID={runID} telemetry={telemetry} />);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: label })));
    expect(fetchMock).toHaveBeenCalledWith(`/api/logs${runID ? `/${runID}` : ""}/${telemetry ? "telemetry" : "export"}`);
    expect(createObjectURL).toHaveBeenCalledWith(blob);
    expect(click).toHaveBeenCalledOnce();
    expect(document.querySelector("a[download]")).toHaveAttribute("download", filename);
    act(() => vi.advanceTimersByTime(1000));
    expect(document.querySelector("a[download]")).toBeNull();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:log-export");
  });

  it("disables the button until the download is ready and reports errors", async () => {
    let resolveResponse!: (value: { ok: boolean }) => void;
    vi.stubGlobal("fetch", vi.fn(() => new Promise((resolve) => { resolveResponse = resolve; })));
    const error = vi.spyOn(NotificationHandler.instance, "error").mockImplementation(() => {});
    render(<LogExportButton runID="run-1" />);
    const button = screen.getByRole("button", { name: "Export JSON" });
    fireEvent.click(button);
    expect(button).toBeDisabled();
    await act(async () => resolveResponse({ ok: false }));
    expect(button).toBeEnabled();
    expect(error).toHaveBeenCalledWith("Unable to export simulation logs. Please try again.");
    expect(createObjectURL).not.toHaveBeenCalled();
  });
});
