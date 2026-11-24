import {
  SimulationActionResponse,
  SimulationConfig,
  SimulationDownlinkRequest,
  SimulationEvent,
  SimulationEventsResponse,
  SimulationRun,
  SimulationRunsResponse,
  SimulationMACCommandRequest,
  SimulationSnapshot,
  SimulationSpeedRequest,
  SimulationUplinkRequest,
} from "@lwn-simulator/contracts";
import { BaseService } from "../../models";
import { GetSimulationEventsOptions } from "./SimulationService.types";

export class SimulationService extends BaseService {
  constructor(baseUrl: string) {
    super("simulation", baseUrl);
  }

  public start = async (config: SimulationConfig): Promise<SimulationSnapshot> =>
    this.apiCaller.post<SimulationSnapshot>("/start", config);

  public setSpeed = async (request: SimulationSpeedRequest): Promise<SimulationSnapshot> =>
    this.apiCaller.post<SimulationSnapshot>("/speed", request);

  public pause = async (): Promise<SimulationSnapshot> =>
    this.apiCaller.post<SimulationSnapshot>("/pause");

  public resume = async (): Promise<SimulationSnapshot> =>
    this.apiCaller.post<SimulationSnapshot>("/resume");

  public stop = async (): Promise<SimulationSnapshot> =>
    this.apiCaller.post<SimulationSnapshot>("/stop");

  public getSnapshot = async (): Promise<SimulationSnapshot> =>
    this.apiCaller.get<SimulationSnapshot>("/snapshot");

  public queueUplink = async (
    req: SimulationUplinkRequest,
  ): Promise<SimulationActionResponse> =>
    this.apiCaller.post<SimulationActionResponse>("/uplinks", req);

  public queueDownlink = async (
    req: SimulationDownlinkRequest,
  ): Promise<SimulationActionResponse> =>
    this.apiCaller.post<SimulationActionResponse>("/downlinks", req);

  public queueMACCommand = async (
    req: SimulationMACCommandRequest,
  ): Promise<SimulationActionResponse> =>
    this.apiCaller.post<SimulationActionResponse>("/mac-commands", req);

  public getEvents = async (
    options: GetSimulationEventsOptions = {},
  ): Promise<SimulationEventsResponse> =>
    this.apiCaller.get<SimulationEventsResponse>("/events", {
      params: options,
    });

  public getMetrics = async (): Promise<string> =>
    this.apiCaller.get<string>("/metrics", { responseType: "text" });

  public getLogExportURL = (id?: string): string =>
    this.apiCaller.buildUrl(id ? `/logs/${encodeURIComponent(id)}/export` : "/logs/export");

  public getTelemetryExportURL = (id?: string): string =>
    this.apiCaller.buildUrl(id ? `/logs/${encodeURIComponent(id)}/telemetry` : "/logs/telemetry");

  public getLogs = async (): Promise<SimulationRunsResponse> =>
    this.apiCaller.get<SimulationRunsResponse>("/logs");

  public getLog = async (id: string): Promise<SimulationRun> =>
    this.apiCaller.get<SimulationRun>(`/logs/${id}`);

  public getLogEvents = async (
    id: string,
    options: GetSimulationEventsOptions = {},
  ): Promise<SimulationEventsResponse> =>
    this.apiCaller.get<SimulationEventsResponse>(`/logs/${id}/events`, {
      params: options,
    });

  public static appendEvent(
    events: Array<SimulationEvent>,
    event: SimulationEvent,
    limit = 1000,
  ): Array<SimulationEvent> {
    return SimulationService.mergeEvents(events, [event], limit);
  }

  public static mergeEvents(
    current: Array<SimulationEvent>,
    incoming: Array<SimulationEvent>,
    limit = 1000,
  ): Array<SimulationEvent> {
    const eventsByID = new Map<string, SimulationEvent>();

    for (const event of [...current, ...incoming]) {
      eventsByID.set(event.id, event);
    }

    return [...eventsByID.values()]
      .sort((left, right) => left.sequence - right.sequence)
      .slice(-limit);
  }
}
