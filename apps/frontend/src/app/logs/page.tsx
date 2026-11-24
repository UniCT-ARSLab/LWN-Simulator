"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import {
  RealtimeWebSocketMessageType,
  RealtimeWebSocketMessage,
  SimulationRun,
  SimulationRunStatus,
} from "@lwn-simulator/contracts";
import {
  useProfileService,
  useSimulationService,
  useWebSocketService,
} from "@lwn-simulator/sdk";
import { Button, ButtonType, Spinner } from "@lwn-simulator/ui-components";
import { LogExportButton } from "@/components/Logs";
import "./logs.scss";

const formatDate = (value: number): string =>
  new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

const formatDuration = (value: number): string => {
  const seconds = Math.floor(value / 1000);
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m ${seconds % 60}s`;
};

const labelStatus = (value: SimulationRunStatus): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

const LogsPage = () => {
  const simulation = useSimulationService();
  const { activeProfileID } = useProfileService();
  const getLogs = simulation.getLogs;
  const { connectionState, subscribe } = useWebSocketService();
  const hasConnectedRef = useRef(false);
  const [runs, setRuns] = useState<Awaited<ReturnType<typeof simulation.getLogs>>["runs"]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    const loadLogs = async () => {
      try {
        const response = await getLogs();
        if (mounted) {
          setRuns(
            [...response.runs].sort(
              (firstRun, secondRun) =>
                secondRun.summary.startedAtMilliseconds -
                firstRun.summary.startedAtMilliseconds,
            ),
          );
          setError(null);
        }
      } catch {
        if (mounted) setError("Unable to load simulation history.");
      } finally {
        if (mounted) setLoading(false);
      }
    };

    void loadLogs();

    return () => {
      mounted = false;
    };
  }, [getLogs]);

  useEffect(() => {
    if (connectionState !== "connected") return;
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true;
      return;
    }

    void getLogs().then((response) => {
      setRuns(
        [...response.runs].sort(
          (firstRun, secondRun) =>
            secondRun.summary.startedAtMilliseconds -
            firstRun.summary.startedAtMilliseconds,
        ),
      );
    }).catch(() => {
      setError("Unable to refresh simulation history.");
    });
  }, [connectionState, getLogs]);

  useEffect(
    () =>
      subscribe((rawMessage) => {
        try {
          const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
          if (message.profileID && message.profileID !== activeProfileID) return;
          if (
            (message.type === RealtimeWebSocketMessageType.SimulationRunCreated ||
              message.type === RealtimeWebSocketMessageType.SimulationRunUpdated) &&
            message.runSummary
          ) {
            setRuns((previous) => {
              const nextRun: SimulationRun = {
                summary: message.runSummary as SimulationRun["summary"],
                objects: previous.find(
                  (run) => run.summary.id === message.runSummary?.id,
                )?.objects ?? [],
              };
              const exists = previous.some(
                (run) => run.summary.id === nextRun.summary.id,
              );
              return [...(exists
                ? previous.map((run) =>
                    run.summary.id === nextRun.summary.id ? nextRun : run,
                  )
                : [...previous, nextRun])].sort(
                (firstRun, secondRun) =>
                  secondRun.summary.startedAtMilliseconds -
                  firstRun.summary.startedAtMilliseconds,
              );
            });
          }
        } catch {
          // Ignore messages owned by other realtime consumers.
        }
      }),
    [activeProfileID, subscribe],
  );

  return (
    <section className="logs-page">
      <header className="logs-page__header">
        <div>
          <span className="eyebrow">Simulation history</span>
          <h1>Logs</h1>
          <p>Review every simulation run and inspect the events observed by each device and gateway.</p>
        </div>
        <div className="logs-page__header-actions">
          <LogExportButton disabled={loading || Boolean(error) || runs.length === 0} />
          <LogExportButton telemetry disabled={loading || Boolean(error) || runs.length === 0} />
          <Link href="/simulation/dashboard">
            <Button value="Open dashboard" type={ButtonType.Primary} />
          </Link>
        </div>
      </header>
      {loading ? (
        <div className="logs-page__loading"><Spinner /></div>
      ) : error ? (
        <p className="logs-page__empty">{error}</p>
      ) : runs.length === 0 ? (
        <div className="logs-page__empty"><h2>No simulation logs yet</h2><p>Start a simulation to create the first run history.</p></div>
      ) : (
        <div className="logs-page__runs">
          {runs.map((run) => (
            <Link className="logs-page__run" href={`/logs/detail?runId=${encodeURIComponent(run.summary.id)}`} key={run.summary.id}>
              <div className="logs-page__run-head">
                <div><span className="eyebrow">Run</span><h2>{formatDate(run.summary.startedAtMilliseconds)}</h2></div>
                <span className={`logs-page__status logs-page__status--${run.summary.status}`}>{labelStatus(run.summary.status)}</span>
              </div>
              <div className="logs-page__metrics">
                <span><strong>{formatDuration(run.summary.durationMilliseconds)}</strong><small>duration</small></span>
                <span><strong>{run.summary.eventCount}</strong><small>events</small></span>
                <span><strong>{run.summary.deviceCount}</strong><small>devices</small></span>
                <span><strong>{run.summary.gatewayCount}</strong><small>gateways</small></span>
                <span><strong>{Math.round(run.summary.packetSuccessRate * 100)}%</strong><small>packet success</small></span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
};

export default LogsPage;
