import { describe, expect, it, vi } from "vitest";
import RootPage from "./page";
import SimulationPage from "./simulation/page";

const { redirect } = vi.hoisted(() => ({ redirect: vi.fn() }));

vi.mock("next/navigation", () => ({
  redirect,
}));

describe("application entry points", () => {
  it("redirects the root page to the simulation section", () => {
    RootPage();

    expect(redirect).toHaveBeenCalledWith("/simulation");
  });

  it("redirects the simulation section to its dashboard", () => {
    SimulationPage();

    expect(redirect).toHaveBeenCalledWith("/simulation/dashboard");
  });
});
