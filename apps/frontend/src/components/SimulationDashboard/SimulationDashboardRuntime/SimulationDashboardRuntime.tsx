"use client";

import { Card, CardLayout } from "@lwn-simulator/ui-components";
import { DeviceRuntimeList } from "../DeviceRuntimeList";
import { GatewayRuntimeList } from "../GatewayRuntimeList";
import { SimulationDashboardRuntimeProps } from "./SimulationDashboardRuntime.types";

export const SimulationDashboardRuntime = (
  props: SimulationDashboardRuntimeProps,
) => {
  return (
    <section className="simulation-dashboard__runtime-grid simulation-dashboard-runtime">
      <Card layout={CardLayout.Padded}>
        <div className="simulation-dashboard__section-heading">
          <div>
            <span className="simulation-dashboard__kicker">DEVICE RUNTIME</span>
            <h2>Live sessions</h2>
          </div>
          <span>{props.devices.length} devices</span>
        </div>
        <DeviceRuntimeList devices={props.devices} deviceProtocols={props.deviceProtocols} />
      </Card>
      <Card layout={CardLayout.Padded}>
        <div className="simulation-dashboard__section-heading">
          <div>
            <span className="simulation-dashboard__kicker">GATEWAY RUNTIME</span>
            <h2>Connectivity</h2>
          </div>
          <span>{props.gateways.length} gateways</span>
        </div>
        <GatewayRuntimeList
          gateways={props.gateways}
          formatState={props.formatState}
        />
      </Card>
    </section>
  );
};
