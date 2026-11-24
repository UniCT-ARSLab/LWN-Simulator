"use client";

import { Card, CardLayout } from "@lwn-simulator/ui-components";
import { SimulationSparkline } from "@/components/SimulationSparkline";
import { SimulationDashboardMetricsProps } from "./SimulationDashboardMetrics.types";

const formatNumber = (value: number): string =>
  new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(value);

export const SimulationDashboardMetrics = (
  props: SimulationDashboardMetricsProps,
) => {
  const packetRate = props.snapshot?.metrics.packetSuccessRate ?? 0;

  return (
    <section className="simulation-dashboard__metric-grid simulation-dashboard-metrics">
      <Card className="simulation-dashboard__metric" layout={CardLayout.Padded}>
        <span className="simulation-dashboard__kicker">
          PACKET SUCCESS RATE
        </span>
        <strong>
          {props.snapshot ? `${(packetRate * 100).toFixed(1)}%` : "—"}
        </strong>
        <SimulationSparkline
          values={props.recentRates}
          label="Packet success rate trend"
        />
        <small>
          {props.snapshot
            ? `${props.snapshot.metrics.successfulUplinks} successful uplinks`
            : "Start traffic to collect metrics"}
        </small>
      </Card>
      <Card className="simulation-dashboard__metric" layout={CardLayout.Padded}>
        <span className="simulation-dashboard__kicker">SIMULATION UPTIME</span>
        <strong>
          {props.snapshot
            ? formatDuration(props.snapshot.state.elapsedMilliseconds)
            : "—"}
        </strong>
        <small>
          {props.snapshot
            ? `${formatNumber(props.snapshot.state.speed)}x clock speed`
            : "Engine idle"}
        </small>
      </Card>
      <Card className="simulation-dashboard__metric" layout={CardLayout.Padded}>
        <span className="simulation-dashboard__kicker">RADIO HEALTH</span>
        <strong>
          {props.snapshot
            ? `${formatNumber(props.snapshot.metrics.averageSNR)} dB`
            : "—"}
        </strong>
        <small>
          {props.snapshot
            ? `${formatNumber(props.snapshot.metrics.averageRSSI)} dBm average RSSI`
            : "Awaiting telemetry"}
        </small>
      </Card>
      <Card className="simulation-dashboard__metric" layout={CardLayout.Padded}>
        <span className="simulation-dashboard__kicker">NETWORK RUNTIME</span>
        <strong>
          {props.snapshot
            ? `${props.snapshot.metrics.activeDevices} / ${props.snapshot.metrics.activeGateways}`
            : "—"}
        </strong>
        <small>active devices / gateways</small>
      </Card>
    </section>
  );
};

const formatDuration = (milliseconds: number): string => {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return `${String(hours).padStart(2, "0")}h ${String(minutes).padStart(2, "0")}m ${String(seconds).padStart(2, "0")}s`;
};
