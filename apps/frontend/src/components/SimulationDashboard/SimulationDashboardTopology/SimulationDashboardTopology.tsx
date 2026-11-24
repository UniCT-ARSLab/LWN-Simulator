"use client";

import { Card, CardLayout } from "@lwn-simulator/ui-components";
import { SimulationEvent } from "@lwn-simulator/contracts";
import { SensorMap } from "@/components/SensorMap";
import { SimulationDashboardTopologyProps } from "./SimulationDashboardTopology.types";
import { formatRawRadioPayload } from "@/utils/radioFrame";

const formatEventType = (value: string): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

const formatDuration = (milliseconds: number): string => {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return `${String(hours).padStart(2, "0")}h ${String(minutes).padStart(2, "0")}m ${String(seconds).padStart(2, "0")}s`;
};

const formatDecodedPayload = (
  payload: SimulationEvent["decodedPayload"],
): string | null => {
  if (!payload) return null;
  try {
    return JSON.stringify(payload);
  } catch {
    return null;
  }
};

export const SimulationDashboardTopology = (
  props: SimulationDashboardTopologyProps,
) => {
  const visibleEvents = [...props.events].reverse().slice(0, 14);

  return (
    <section className="simulation-dashboard__main-grid simulation-dashboard-topology">
      <Card className="simulation-dashboard__map" layout={CardLayout.Padded}>
        <div className="simulation-dashboard__section-heading">
          <div>
            <span className="simulation-dashboard__kicker">LIVE TOPOLOGY</span>
            <h2>Propagation map</h2>
          </div>
          <span>
            {props.devices.length} devices · {props.gateways.length} gateways
          </span>
        </div>
        <div className="simulation-dashboard__map-canvas">
          <SensorMap
            logic={props.mapLogic}
            devices={props.devices}
            gateways={props.gateways}
            links={props.links}
          />
        </div>
      </Card>
      <Card
        className="simulation-dashboard__events"
        layout={CardLayout.Padded}
      >
        <div className="simulation-dashboard__section-heading">
          <div>
            <span className="simulation-dashboard__kicker">
              REAL-TIME EVENT STREAM
            </span>
            <h2>Engine activity</h2>
          </div>
          <span>{props.events.length} events</span>
        </div>
        <div className="simulation-dashboard__event-list">
          {visibleEvents.length === 0 && (
            <p className="simulation-dashboard__empty">
              Events will appear when the engine starts.
            </p>
          )}
          {visibleEvents.map((event) => {
            const decodedPayload = formatDecodedPayload(event.decodedPayload);
            const rawPayload = formatRawRadioPayload(event.radioFrame);
            return (
              <div className="simulation-dashboard__event" key={event.id}>
                <time>{formatDuration(event.timestampMilliseconds)}</time>
                <strong>{formatEventType(event.type)}</strong>
                <span title={rawPayload ?? decodedPayload ?? undefined}>
                  {event.message}
                  {decodedPayload ? ` · Decoded ${decodedPayload}` : ""}
                  {rawPayload ? ` · ${rawPayload}` : ""}
                  {event.error ? ` · Error: ${event.error}` : ""}
                </span>
              </div>
            );
          })}
        </div>
      </Card>
    </section>
  );
};
