import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  SimulationActionResponse,
  SimulationConfig,
  SimulationDownlinkRequest,
  SimulationEvent,
  SimulationEventsResponse,
  SimulationMACCommandRequest,
  SimulationSnapshot,
  SimulationUplinkRequest,
  RealtimeWebSocketMessage,
} from "@lwn-simulator/contracts";
import { useWebSocketService } from "../WebSocketService";
import SimulationServiceContext from "./SimulationServiceContext";
import {
  GetSimulationEventsOptions,
  SimulationConnectionState,
  SimulationServiceContent,
  SimulationServiceProviderProps,
} from "./SimulationService.types";
import { SimulationService } from "./SimulationService";
import { RealtimeWebSocketMessageType } from "../../models/RealtimeWebSocket";

const realtimeRenderIntervalMilliseconds = 100;
const SimulationServiceProvider = (props: SimulationServiceProviderProps) => {
  // States
  const [snapshot, setSnapshot] = useState<SimulationSnapshot | null>(null);
  const [events, setEvents] = useState<Array<SimulationEvent>>([]);
  const [error, setError] = useState<Error | null>(null);
  const [clockNowMilliseconds, setClockNowMilliseconds] = useState(0);

  // Hooks
  const {
    subscribe: subscribeWebSocket,
    connectionState,
    error: websocketError,
  } = useWebSocketService();
  const eventsRef = useRef<Array<SimulationEvent>>([]);
  const pendingSnapshotRef = useRef<SimulationSnapshot | null>(null);
  const pendingEventsRef = useRef<Array<SimulationEvent>>([]);
  const renderTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hasSnapshotRef = useRef(false);
  const [snapshotClock, setSnapshotClock] = useState<{
    elapsedMilliseconds: number;
    receivedAtMilliseconds: number;
    speed: number;
  } | null>(null);

  // Memos
  const service = useMemo(
    () => new SimulationService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const updateSnapshot = useCallback((next: SimulationSnapshot) => {
    hasSnapshotRef.current = true;
    setSnapshotClock({
      elapsedMilliseconds: next.state.elapsedMilliseconds,
      receivedAtMilliseconds: Date.now(),
      speed: next.state.speed,
    });
    setSnapshot(next);
  }, []);

  const updateEvents = useCallback((next: Array<SimulationEvent>) => {
    eventsRef.current = next;
    setEvents(next);
  }, []);

  const flushRealtimeUpdates = useCallback(() => {
    renderTimerRef.current = null;

    if (pendingSnapshotRef.current) {
      updateSnapshot(pendingSnapshotRef.current);
      pendingSnapshotRef.current = null;
    }

    if (pendingEventsRef.current.length > 0) {
      updateEvents(
        SimulationService.mergeEvents(
          eventsRef.current,
          pendingEventsRef.current,
        ),
      );
      pendingEventsRef.current = [];
    }
  }, [updateEvents, updateSnapshot]);

  const scheduleRealtimeRender = useCallback(() => {
    if (renderTimerRef.current) return;

    renderTimerRef.current = setTimeout(
      flushRealtimeUpdates,
      realtimeRenderIntervalMilliseconds,
    );
  }, [flushRealtimeUpdates]);

  const getEvents = useCallback(
    async (
      options: GetSimulationEventsOptions = {},
    ): Promise<SimulationEventsResponse> => {
      const lastEvent = eventsRef.current.at(-1);
      const response = await service.getEvents({
        afterSequence: options.afterSequence ?? lastEvent?.sequence ?? 0,
        limit: options.limit ?? 100,
      });
      updateEvents(
        SimulationService.mergeEvents(eventsRef.current, response.events),
      );
      setError(null);
      return response;
    },
    [service, updateEvents],
  );

  const handleWebSocketMessage = useCallback(
    (rawMessage: string) => {
      try {
        const update = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
        if (update.profileID && update.profileID !== props.profileID) return;
        if (update.type === RealtimeWebSocketMessageType.Error) {
          setError(new Error(update.error ?? "Realtime websocket error"));
          return;
        }
        if (update.snapshot) {
          pendingSnapshotRef.current = update.snapshot;
        }
        if (
          update.type === RealtimeWebSocketMessageType.SimulationEvent &&
          update.event
        ) {
          pendingEventsRef.current.push(update.event);
        }
        scheduleRealtimeRender();
      } catch (parseError) {
        setError(
          parseError instanceof Error
            ? parseError
            : new Error("Invalid realtime websocket message"),
        );
      }
    },
    [props.profileID, scheduleRealtimeRender],
  );

  // Effects
  useEffect(
    () => subscribeWebSocket(handleWebSocketMessage),
    [handleWebSocketMessage, props.profileID, subscribeWebSocket],
  );

  useEffect(() => {
    if (connectionState !== "connected" || !hasSnapshotRef.current) return;

    void getEvents().catch((eventsError: unknown) => {
      setError(
        eventsError instanceof Error
          ? eventsError
          : new Error("Unable to recover simulation events"),
      );
    });
  }, [connectionState, getEvents]);

  const disconnect = useCallback(() => {
    if (renderTimerRef.current) {
      clearTimeout(renderTimerRef.current);
      flushRealtimeUpdates();
    }
  }, [flushRealtimeUpdates]);

  const connect = useCallback(() => undefined, []);

  const { onControlStatus, profileID } = props;
  const applyControlSnapshot = useCallback((next: SimulationSnapshot) => {
    pendingSnapshotRef.current = null;
    setError(null);
    updateSnapshot(next);
    onControlStatus?.(profileID, next.state.status);
  }, [onControlStatus, profileID, updateSnapshot]);

  const start = useCallback(
    async (config: SimulationConfig) => {
      const next = await service.start(config);
      pendingEventsRef.current = [];
      updateEvents([]);
      applyControlSnapshot(next);
      return next;
    },
    [applyControlSnapshot, service, updateEvents],
  );

  const pause = useCallback(async () => {
    const next = await service.pause();
    applyControlSnapshot(next);
    return next;
  }, [applyControlSnapshot, service]);

  const resume = useCallback(async () => {
    const next = await service.resume();
    applyControlSnapshot(next);
    return next;
  }, [applyControlSnapshot, service]);

  const stop = useCallback(async () => {
    const next = await service.stop();
    applyControlSnapshot(next);
    disconnect();
    return next;
  }, [applyControlSnapshot, disconnect, service]);

  const getSnapshot = useCallback(async () => {
    const next = await service.getSnapshot();
    updateSnapshot(next);
    setError(null);
    return next;
  }, [service, updateSnapshot]);

  const queueUplink = useCallback(
    async (
      req: SimulationUplinkRequest,
    ): Promise<SimulationActionResponse> => {
      const response = await service.queueUplink(req);
      setError(null);
      return response;
    },
    [service],
  );

  const queueDownlink = useCallback(
    async (
      req: SimulationDownlinkRequest,
    ): Promise<SimulationActionResponse> => {
      const response = await service.queueDownlink(req);
      setError(null);
      return response;
    },
    [service],
  );

  const queueMACCommand = useCallback(
    async (
      req: SimulationMACCommandRequest,
    ): Promise<SimulationActionResponse> => {
      const response = await service.queueMACCommand(req);
      setError(null);
      return response;
    },
    [service],
  );

  const getMetrics = useCallback(async (): Promise<string> => {
    const response = await service.getMetrics();
    setError(null);
    return response;
  }, [service]);

  const getLogExportURL = useCallback((id?: string) => service.getLogExportURL(id), [service]);
  const getLogs = useCallback(() => service.getLogs(), [service]);
  const getLog = useCallback((id: string) => service.getLog(id), [service]);
  const getLogEvents = useCallback(
    (id: string, options: GetSimulationEventsOptions = {}) =>
      service.getLogEvents(id, options),
    [service],
  );

  const setSpeed = useCallback(
    async (request: Parameters<SimulationService["setSpeed"]>[0]) => {
      const response = await service.setSpeed(request);
      updateSnapshot(response);
      setError(null);
      return response;
    },
    [service, updateSnapshot],
  );

  useEffect(() => disconnect, [disconnect]);

  useEffect(() => {
    if (snapshot?.state.status !== "running") return undefined;

    const timer = setInterval(() => setClockNowMilliseconds(Date.now()), 100);
    return () => clearInterval(timer);
  }, [snapshot?.state.status]);

  const displayedSnapshot = useMemo(() => {
    if (!snapshot || snapshot.state.status !== "running") return snapshot;

    const clock = snapshotClock;
    if (!clock) return snapshot;

    const elapsedMilliseconds = Math.max(
      snapshot.state.elapsedMilliseconds,
      clock.elapsedMilliseconds +
        (clockNowMilliseconds - clock.receivedAtMilliseconds) * clock.speed,
    );
    if (elapsedMilliseconds === snapshot.state.elapsedMilliseconds) {
      return snapshot;
    }

    return {
      ...snapshot,
      state: {
        ...snapshot.state,
        elapsedMilliseconds,
      },
    };
  }, [clockNowMilliseconds, snapshot, snapshotClock]);

  // Memos
  const value = useMemo(
    (): SimulationServiceContent => ({
      start,
      setSpeed,
      pause,
      resume,
      stop,
      getSnapshot,
      queueUplink,
      queueDownlink,
      queueMACCommand,
      getEvents,
      getMetrics,
      getLogExportURL,
      getLogs,
      getLog,
      getLogEvents,
      connect,
      disconnect,
      snapshot: displayedSnapshot,
      events,
      connectionState: connectionState as SimulationConnectionState,
      error: error ?? websocketError,
    }),
    [
      start,
      setSpeed,
      pause,
      resume,
      stop,
      getSnapshot,
      queueUplink,
      queueDownlink,
      queueMACCommand,
      getEvents,
      getMetrics,
      getLogExportURL,
      getLogs,
      getLog,
      getLogEvents,
      connect,
      disconnect,
      displayedSnapshot,
      events,
      connectionState,
      websocketError,
      error,
    ],
  );

  return (
    <SimulationServiceContext.Provider value={value}>
      {props.children}
    </SimulationServiceContext.Provider>
  );
};

export default SimulationServiceProvider;
