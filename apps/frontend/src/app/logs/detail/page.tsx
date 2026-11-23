"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  RealtimeWebSocketMessageType,
  RealtimeWebSocketMessage,
  SimulationEvent,
  SimulationLogObject,
  SimulationRun,
} from "@lwn-simulator/contracts";
import {
  useSimulationService,
  useWebSocketService,
} from "@lwn-simulator/sdk";
import { Button, Spinner } from "@lwn-simulator/ui-components";
import { LogExportButton, VirtualEventList, VirtualObjectList } from "@/components/Logs";
import "../logs.scss";

const eventPageSize = 1000;
const maxLiveEvents = 5000;

const formatDate = (value: number): string =>
  new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

const formatEventType = (value: string): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

const mergeEvents = (
  current: SimulationEvent[],
  incoming: SimulationEvent[],
  maxEvents?: number,
): SimulationEvent[] => {
  const bySequence = new Map<number, SimulationEvent>();
  [...current, ...incoming].forEach((event) => bySequence.set(event.sequence, event));
  const merged = [...bySequence.values()].sort((left, right) => left.sequence - right.sequence);
  return maxEvents && merged.length > maxEvents ? merged.slice(-maxEvents) : merged;
};

const LogDetailsContent = () => {
  const simulation = useSimulationService();
  const getLog = simulation.getLog;
  const getLogEvents = simulation.getLogEvents;
  const { connectionState, subscribe } = useWebSocketService();
  const hasConnectedRef = useRef(false);
  const runID = useSearchParams().get("runId");
  const [run, setRun] = useState<SimulationRun | null>(null);
  const runRef = useRef<SimulationRun | null>(null);
  const [events, setEvents] = useState<SimulationEvent[]>([]);
  const eventsRef = useRef<SimulationEvent[]>([]);
  const latestSequenceRef = useRef(0);
  const [selectedObject, setSelectedObject] = useState<SimulationLogObject | null>(null);
  const selectedObjectRef = useRef<SimulationLogObject | null>(null);
  const [objectEvents, setObjectEvents] = useState<SimulationEvent[]>([]);
  const objectEventsRef = useRef<SimulationEvent[]>([]);
  const objectLatestSequenceRef = useRef(0);
  const [objectLoading, setObjectLoading] = useState(false);
  const [loadingOlderObjectEvents, setLoadingOlderObjectEvents] = useState(false);
  const [objectFilter, setObjectFilter] = useState("");
  const [eventListNearEnd, setEventListNearEnd] = useState(true);
  const [loadingOlderEvents, setLoadingOlderEvents] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    selectedObjectRef.current = selectedObject;
  }, [selectedObject]);

  const updateEvents = useCallback((incoming: SimulationEvent[], maxEvents?: number) => {
    setEvents((current) => {
      const merged = mergeEvents(current, incoming, maxEvents);
      eventsRef.current = merged;
      return merged;
    });
  }, []);

  useEffect(() => {
    if (!runID) {
      setError("No simulation run was selected.");
      setLoading(false);
      return;
    }

    let mounted = true;
    const loadInitialState = async () => {
      try {
        const loadedRun = await getLog(runID);
        const loadedEvents = await getLogEvents(runID, {
          beforeSequence: loadedRun.summary.eventCount + 1,
          limit: eventPageSize,
        });
        if (!mounted) return;
        runRef.current = loadedRun;
        setRun(loadedRun);
        latestSequenceRef.current = Math.max(
          latestSequenceRef.current,
          loadedEvents.lastSequence,
        );
        updateEvents(
          loadedEvents.events,
          loadedRun.summary.status === "running" || loadedRun.summary.status === "paused"
            ? maxLiveEvents
            : undefined,
        );
        setSelectedObject((currentObject) => currentObject ?? loadedRun.objects[0] ?? null);
        setError(null);
        setLoading(false);
      } catch {
        if (mounted) {
          setError("Unable to load this simulation log.");
          setLoading(false);
        }
      }
    };

    void loadInitialState();
    return () => {
      mounted = false;
    };
  }, [getLog, getLogEvents, runID, updateEvents]);

  useEffect(() => {
    if (!runID || connectionState !== "connected") return;
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true;
      return;
    }

    let mounted = true;
    const recoverAfterReconnect = async () => {
      try {
        const [loadedRun, loadedEvents] = await Promise.all([
          getLog(runID),
          getLogEvents(runID, {
            afterSequence: latestSequenceRef.current,
            limit: eventPageSize,
          }),
        ]);
        if (!mounted) return;
        runRef.current = loadedRun;
        setRun(loadedRun);
        latestSequenceRef.current = Math.max(
          latestSequenceRef.current,
          loadedEvents.lastSequence,
        );
        updateEvents(loadedEvents.events, maxLiveEvents);
        const selected = selectedObjectRef.current;
        if (selected) {
          const objectEventsFromRecovery = loadedEvents.events.filter((event) =>
            selected.kind === "device"
              ? event.deviceID === selected.id
              : event.gatewayID === selected.id,
          );
          const merged = mergeEvents(objectEventsRef.current, objectEventsFromRecovery);
          objectEventsRef.current = merged;
          setObjectEvents(merged);
        }
      } catch {
        if (mounted) setError("Unable to recover the realtime log stream.");
      }
    };

    void recoverAfterReconnect();
    return () => {
      mounted = false;
    };
  }, [connectionState, getLog, getLogEvents, runID, updateEvents]);

  useEffect(() => {
    if (!runID) return;

    return subscribe((rawMessage) => {
      try {
        const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
        if (message.runID !== runID) return;

        if (
          message.type === RealtimeWebSocketMessageType.SimulationRunUpdated &&
          message.runSummary
        ) {
          setRun((current) => {
            if (!current) return current;
            const next = { ...current, summary: message.runSummary as SimulationRun["summary"] };
            runRef.current = next;
            return next;
          });
        }

        if (
          message.type === RealtimeWebSocketMessageType.SimulationEvent &&
          message.event
        ) {
          latestSequenceRef.current = Math.max(
            latestSequenceRef.current,
            message.event.sequence,
          );
          updateEvents([message.event], maxLiveEvents);
          if (
            selectedObjectRef.current &&
            ((selectedObjectRef.current.kind === "device" &&
              message.event.deviceID === selectedObjectRef.current.id) ||
              (selectedObjectRef.current.kind === "gateway" &&
                message.event.gatewayID === selectedObjectRef.current.id))
          ) {
            const nextObjectEvents = mergeEvents(objectEventsRef.current, [message.event]);
            objectEventsRef.current = nextObjectEvents;
            objectLatestSequenceRef.current = Math.max(
              objectLatestSequenceRef.current,
              message.event.sequence,
            );
            setObjectEvents(nextObjectEvents);
            setObjectLoading(false);
          }
        }
      } catch {
        // Ignore messages owned by another realtime consumer.
      }
    });
  }, [runID, subscribe, updateEvents]);

  const loadOlderEvents = useCallback(async () => {
    if (!runID || loadingOlderEvents || eventsRef.current.length === 0) return;
    const oldestSequence = eventsRef.current[0].sequence;
    if (oldestSequence <= 1) return;
    setLoadingOlderEvents(true);
    try {
      const response = await getLogEvents(runID, {
        beforeSequence: oldestSequence,
        limit: eventPageSize,
      });
      const merged = mergeEvents(response.events, eventsRef.current);
      eventsRef.current = merged;
      setEvents(merged);
    } finally {
      setLoadingOlderEvents(false);
    }
  }, [getLogEvents, loadingOlderEvents, runID]);

  const filteredObjects = useMemo(() => {
    const normalized = objectFilter.trim().toLowerCase();
    if (!run || normalized.length === 0) return run?.objects ?? [];
    return run.objects.filter((object) =>
      `${object.name} ${object.identifier} ${object.kind}`.toLowerCase().includes(normalized),
    );
  }, [objectFilter, run]);

  const loadOlderObjectEvents = useCallback(async () => {
    if (!runID || !selectedObject || loadingOlderObjectEvents || objectEventsRef.current.length === 0) return;
    const oldestSequence = objectEventsRef.current[0].sequence;
    if (oldestSequence <= 1) return;
    setLoadingOlderObjectEvents(true);
    try {
      const filter = selectedObject.kind === "device"
        ? { deviceID: selectedObject.id }
        : { gatewayID: selectedObject.id };
      const response = await getLogEvents(runID, {
        ...filter,
        beforeSequence: oldestSequence,
        limit: eventPageSize,
      });
      const merged = mergeEvents(response.events, objectEventsRef.current);
      objectEventsRef.current = merged;
      setObjectEvents(merged);
    } finally {
      setLoadingOlderObjectEvents(false);
    }
  }, [getLogEvents, loadingOlderObjectEvents, runID, selectedObject]);

  useEffect(() => {
    if (!runID || !runRef.current || !selectedObject) return;
    let mounted = true;
    const filter = selectedObject.kind === "device"
      ? { deviceID: selectedObject.id }
      : { gatewayID: selectedObject.id };

    const loadObjectEvents = async () => {
      try {
        const response = await getLogEvents(runID, {
          ...filter,
          beforeSequence: (runRef.current?.summary.eventCount ?? 0) + 1,
          limit: eventPageSize,
        });
        if (!mounted) return;
        const merged = mergeEvents(objectEventsRef.current, response.events);
        objectEventsRef.current = merged;
        objectLatestSequenceRef.current = Math.max(
          objectLatestSequenceRef.current,
          response.lastSequence,
        );
        setObjectEvents(merged);
        setObjectLoading(false);
      } catch {
        if (mounted) setObjectLoading(false);
      }
    };

    objectEventsRef.current = [];
    objectLatestSequenceRef.current = 0;
    selectedObjectRef.current = selectedObject;
    setObjectEvents([]);
    setObjectLoading(true);
    void loadObjectEvents();
    return () => {
      mounted = false;
    };
  }, [getLogEvents, runID, selectedObject]);

  if (loading) return <div className="logs-page__loading"><Spinner /></div>;

  if (error || !run) {
    return <section className="logs-page"><p className="logs-page__empty">{error ?? "Simulation log not found."}</p></section>;
  }

  return (
    <section className="logs-page logs-page--detail">
      <header className="logs-page__header">
        <div>
          <span className="eyebrow">Simulation run</span>
          <h1>{formatDate(run.summary.startedAtMilliseconds)}</h1>
          <p>{run.summary.eventCount} recorded events · {run.summary.deviceCount} devices · {run.summary.gatewayCount} gateways</p>
        </div>
        <div className="logs-page__header-actions">
          <span className={`logs-page__status logs-page__status--${run.summary.status}`}>{formatEventType(run.summary.status)}</span>
          <LogExportButton runID={run.summary.id} />
          <LogExportButton telemetry runID={run.summary.id} />
          <Link href="/logs"><Button value="Back" /></Link>
        </div>
      </header>
      <div className="logs-page__detail-grid">
        <div className="logs-page__objects-column">
          <section className="logs-page__panel logs-page__panel--objects">
            <div className="logs-page__panel-head">
              <div><span className="eyebrow">Object history</span><h2>Devices and gateways</h2></div>
              <input
                aria-label="Search devices and gateways"
                className="logs-page__object-filter"
                onChange={(event) => setObjectFilter(event.target.value)}
                placeholder="Search objects"
                value={objectFilter}
              />
            </div>
            <VirtualObjectList objects={filteredObjects} onSelect={setSelectedObject} selectedObject={selectedObject} />
          </section>
          <section className="logs-page__panel logs-page__panel--object-events">
            <div className="logs-page__panel-head"><span className="eyebrow">Selected object</span><h2>{selectedObject?.name ?? "Object events"}</h2></div>
            {objectLoading ? <Spinner /> : (
              <>
                {loadingOlderObjectEvents ? <small className="logs-page__loading-label">Loading older events…</small> : null}
                <VirtualEventList
                  autoScroll={eventListNearEnd}
                  events={objectEvents}
                  emptyMessage="No events for this object."
                  onReachStart={() => void loadOlderObjectEvents()}
                />
              </>
            )}
          </section>
        </div>
        <section className="logs-page__panel logs-page__panel--events">
          <span className="eyebrow">Event stream</span>
          <h2>All recorded events</h2>
          {loadingOlderEvents ? <small className="logs-page__loading-label">Loading older events…</small> : null}
          <VirtualEventList
            autoScroll={eventListNearEnd}
            events={events}
            emptyMessage="No events were recorded."
            onNearEnd={setEventListNearEnd}
            onReachStart={() => void loadOlderEvents()}
          />
        </section>
      </div>
    </section>
  );
};

const LogDetailsPage = () => (
  <Suspense fallback={<div className="logs-page__loading"><Spinner /></div>}>
    <LogDetailsContent />
  </Suspense>
);

export default LogDetailsPage;
