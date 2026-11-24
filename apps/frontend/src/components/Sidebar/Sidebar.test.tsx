import { render, screen } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SimulationStatus } from "@lwn-simulator/contracts";
import Sidebar from "./Sidebar";

const { usePathname, useSimulationService, useProfileService } = vi.hoisted(
  () => ({
    usePathname: vi.fn(),
    useSimulationService: vi.fn(),
    useProfileService: vi.fn(),
  }),
);

vi.mock("next/navigation", () => ({
  usePathname,
}));

vi.mock("next/link", () => ({
  default: ({ children, ...props }: PropsWithChildren<Record<string, unknown>>) => (
    <a {...props}>{children}</a>
  ),
}));

vi.mock("@lwn-simulator/sdk", () => ({
  useSimulationService,
  useProfileService,
}));

const renderSidebar = () =>
  render(
    <Sidebar>
      <div>Page content</div>
    </Sidebar>,
  );

describe("Sidebar", () => {
  beforeEach(() => {
    useSimulationService.mockReturnValue({ snapshot: null });
    useProfileService.mockReturnValue({ simulationActivity: null });
  });

  it("keeps Logs active for a simulation run detail URL", () => {
    usePathname.mockReturnValue("/logs/run-123");
    renderSidebar();

    const logsLink = screen.getByRole("link", { name: /Logs/ });
    expect(logsLink).toHaveClass("active");
    expect(logsLink).toHaveAttribute("href", "/logs");
  });

  it("shows active simulation status from the shared simulation state", () => {
    usePathname.mockReturnValue("/hardware/devices");
    useSimulationService.mockReturnValue({
      snapshot: { state: { status: SimulationStatus.Running } },
    });
    renderSidebar();

    expect(screen.getByText("Simulation active")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /devices/ })).toHaveClass("active");
  });

  it("shows inactive status when there is no active simulation", () => {
    usePathname.mockReturnValue("/hardware/gateways");
    renderSidebar();

    expect(screen.getByText("Simulation inactive")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /gateways/ })).toHaveClass("active");
  });
});
