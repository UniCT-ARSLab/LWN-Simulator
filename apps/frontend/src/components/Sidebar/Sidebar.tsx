"use client";

import { SidebarProps } from "./Sidebar.types";
import { usePathname } from "next/navigation";
import cn from "classnames";
import Link from "next/link";
import { SimulationStatus } from "@lwn-simulator/contracts";
import { useProfileService, useSimulationService } from "@lwn-simulator/sdk";

const Sidebar = (props: SidebarProps) => {
  const routes: Record<
    string,
    Array<{ route: string; icon: string; label?: string }>
  > = {
    simulation: [{ route: "dashboard", icon: "dashboard" }],
    hardware: [
      { route: "devices", icon: "sensors" },
      { route: "gateways", icon: "router" },
      {
        route: "gateway-bridge",
        icon: "settings_ethernet",
        label: "Gateway Bridge",
      },
    ],
    logs: [{ route: "", icon: "history", label: "Logs" }],
  };

  // Hooks
  const pathname = usePathname();
  const simulationService = useSimulationService();
  const profileService = useProfileService();
  const simulationStatus = simulationService.snapshot?.state.status;
  const simulationIsActive =
    profileService.simulationActivity?.active === true ||
    simulationStatus === SimulationStatus.Running ||
    simulationStatus === SimulationStatus.Paused;

  const mainSection = pathname?.split("/")[1] || "";
  const subSection = pathname?.split("/")[2] || "";
  const isLogsSection = mainSection === "logs";

  const isHome = mainSection === "";
  const currentRoutes = isHome ? routes.simulation : routes[mainSection] || [];
  const currentSubSection = isHome ? "dashboard" : subSection;

  return (
    <div className="sidebar">
      <aside>
        <div className="status">
          <span className="material-symbols-outlined icon">vital_signs</span>
          {simulationIsActive ? (
            <div className="active">Simulation active</div>
          ) : (
            <div className="inactive">Simulation inactive</div>
          )}
        </div>

        {currentRoutes.map((route, i) => {
          return (
            <Link
              className={cn(
                "sub-section",
                (isLogsSection || currentSubSection === route.route) && "active",
              )}
              key={i}
              href={route.route ? `/${mainSection}/${route.route}` : `/${mainSection}`}
            >
              <span className="material-symbols-outlined icon">
                {route.icon}
              </span>
              <span>{route.label ?? route.route}</span>
            </Link>
          );
        })}

        <hr />
        <a
          href="https://giuseppe-compagnone.github.io/LWN-Simulator/docs/"
          target="_blank"
          className="sub-section"
          rel="noreferrer"
        >
          <span className="material-symbols-outlined icon">menu_book</span>
          <span>Documentation</span>
        </a>
      </aside>
      <main className="content">{props.children}</main>
    </div>
  );
};

export default Sidebar;
