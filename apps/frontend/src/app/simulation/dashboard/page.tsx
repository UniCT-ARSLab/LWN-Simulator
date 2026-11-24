"use client";

import {
  FormValue,
  NotificationHandler,
  Spinner,
  rangeField,
  textField,
} from "@lwn-simulator/ui-components";
import {
  GatewayType,
  SimulationEventType,
  SimulationStatus,
} from "@lwn-simulator/contracts";
import {
  useDeviceService,
  useGatewayService,
  useProfileConfigurationService,
  useProfileService,
  useSimulationService,
} from "@lwn-simulator/sdk";
import {
  SimulationCommandPanel,
  SimulationDashboardControls,
  SimulationDashboardMetrics,
  SimulationDashboardRuntime,
  SimulationDashboardTopology,
  SensorMapLink,
  useSensorMap,
} from "@/components";
import { useCallback, useEffect, useMemo, useState } from "react";

const formatEventType = (value: string): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

const distanceMeters = (
  latitudeA: number,
  longitudeA: number,
  latitudeB: number,
  longitudeB: number,
): number => {
  const earthRadiusMeters = 6_371_000;
  const toRadians = (degrees: number): number => (degrees * Math.PI) / 180;
  const latitudeDelta = toRadians(latitudeB - latitudeA);
  const longitudeDelta = toRadians(longitudeB - longitudeA);
  const latitudeARadians = toRadians(latitudeA);
  const latitudeBRadians = toRadians(latitudeB);
  const haversine =
    Math.sin(latitudeDelta / 2) ** 2 +
    Math.cos(latitudeARadians) *
      Math.cos(latitudeBRadians) *
      Math.sin(longitudeDelta / 2) ** 2;

  return (
    2 *
    earthRadiusMeters *
    Math.asin(Math.sqrt(Math.min(1, Math.max(0, haversine))))
  );
};

const communicationEventTypes = new Set<SimulationEventType>([
  SimulationEventType.GatewayPacketReceived,
  SimulationEventType.JoinRequestReceived,
  SimulationEventType.DeviceUplinkTransmitted,
  SimulationEventType.DeviceDownlinkTransmitted,
  SimulationEventType.GatewayPacketIngress,
  SimulationEventType.GatewayPacketEgress,
  SimulationEventType.ClassBDownlinkTransmitted,
]);

const radioTransmissionEventTypes = new Set<SimulationEventType>([
  SimulationEventType.RadioTransmissionStarted,
]);

const linkActivityWindowMilliseconds = 15_000;

const SimulationDashboardPage = () => {
  const simulation = useSimulationService();
  const profileService = useProfileService();
  const profileConfiguration = useProfileConfigurationService();
  const deviceService = useDeviceService();
  const gatewayService = useGatewayService();
  const devices = useMemo(
    () => deviceService.devices ?? [],
    [deviceService.devices],
  );
  const deviceProtocols = useMemo(
    () => Object.fromEntries(devices.map((device) => [device.id, device.radioProtocol])),
    [devices],
  );
  const gateways = useMemo(
    () => gatewayService.gateways ?? [],
    [gatewayService.gateways],
  );
  const hardwareLoaded = deviceService.devices !== null && gatewayService.gateways !== null;
  const hasActiveHardware =
    devices.some((device) => device.active) ||
    gateways.some((gateway) => gateway.active);
  const requiresRealtime =
    profileConfiguration.gatewayBridgeConfig?.enabled === true ||
    gateways.some((gateway) => gateway.type === GatewayType.Real);
  const mapLogic = useSensorMap({});
  const [controlError, setControlError] = useState<string | null>(null);
  const [isLoadingSnapshot, setIsLoadingSnapshot] = useState(true);
  const getSnapshot = simulation.getSnapshot;
  const getEvents = simulation.getEvents;

  useEffect(() => {
    let mounted = true;

    void getSnapshot()
      .then(() => getEvents({ afterSequence: 0, limit: 1000 }))
      .catch(() => undefined)
      .finally(() => {
        if (mounted) setIsLoadingSnapshot(false);
      });

    return () => {
      mounted = false;
    };
  }, [getEvents, getSnapshot]);

  const snapshot = simulation.snapshot;
  const status = snapshot?.state.status ?? SimulationStatus.Idle;
  const isRunning = status === SimulationStatus.Running;
  const isPaused = status === SimulationStatus.Paused;
  const canStartFromCurrentProfile =
    status === SimulationStatus.Idle ||
    status === SimulationStatus.Stopped ||
    status === SimulationStatus.Failed;
  const hasActiveSimulation = profileService.simulationActivity?.active ?? false;
  const activeSimulationIsElsewhere =
    hasActiveSimulation &&
    profileService.simulationActivity?.profileID !== undefined &&
    profileService.simulationActivity.profileID !==
      profileService.activeProfileID;
  const canStart =
    canStartFromCurrentProfile &&
    hardwareLoaded &&
    hasActiveHardware &&
    !profileService.simulationActivityLoading &&
    !hasActiveSimulation;
  const activeSimulationProfile = profileService.profiles.find(
    (profile) =>
      profile.id === profileService.simulationActivity?.profileID,
  );
  const packetRate = snapshot?.metrics.packetSuccessRate ?? 0;
  const simulationEvents = simulation.events;
  const recentRates = useMemo(
    () =>
      simulationEvents
        .filter((event) => event.type === "metrics-updated")
        .map(() => packetRate * 100)
        .slice(-24),
    [packetRate, simulationEvents],
  );
  const mapLinks = useMemo<Array<SensorMapLink>>(() => {
    const devicesByID = new Map(devices.map((device) => [device.id, device]));
    const gatewaysByID = new Map(
      gateways.map((gateway) => [gateway.id, gateway]),
    );
    const latestLinks = new Map<string, SensorMapLink & { sequence: number }>();
    const latestSimulationStartSequence = simulationEvents.reduce(
      (sequence, event) =>
        event.type === SimulationEventType.SimulationStarted
          ? Math.max(sequence, event.sequence)
          : sequence,
      0,
    );
    const elapsedMilliseconds = snapshot?.state.elapsedMilliseconds ?? 0;
    const minimumLinkTimestamp = Math.max(
      0,
      elapsedMilliseconds - linkActivityWindowMilliseconds,
    );

    const registerLink = (
      device: (typeof devices)[number],
      gateway: (typeof gateways)[number],
      sequence: number,
    ) => {
      if (!gateway.active) return;

      const distance = distanceMeters(
        device.locationConfig.latitude,
        device.locationConfig.longitude,
        gateway.latitude,
        gateway.longitude,
      );
      if (distance > device.advancedConfig.antennaRange) return;

      const id = `${device.id}:${gateway.id}`;
      const previous = latestLinks.get(id);
      if (previous && previous.sequence >= sequence) return;

      latestLinks.set(id, {
        id,
        sequence,
        from: {
          latitude: device.locationConfig.latitude,
          longitude: device.locationConfig.longitude,
        },
        to: {
          latitude: gateway.latitude,
          longitude: gateway.longitude,
        },
      });
    };

    for (const event of simulationEvents) {
      if (
        event.sequence < latestSimulationStartSequence ||
        event.timestampMilliseconds < minimumLinkTimestamp
      ) {
        continue;
      }

      if (!event.deviceID) continue;

      const device = devicesByID.get(event.deviceID);
      if (!device) continue;

      if (communicationEventTypes.has(event.type) && event.gatewayID) {
        const gateway = gatewaysByID.get(event.gatewayID);
        if (gateway) registerLink(device, gateway, event.sequence);
        continue;
      }

      if (!radioTransmissionEventTypes.has(event.type)) continue;

      for (const gateway of gateways) {
        if (gateway.type !== GatewayType.Virtual) continue;

        registerLink(device, gateway, event.sequence);
      }
    }

    return [...latestLinks.values()].map((link) => ({
      id: link.id,
      from: link.from,
      to: link.to,
    }));
  }, [devices, gateways, simulationEvents, snapshot?.state.elapsedMilliseconds]);

  const runControl = async (
    action: () => Promise<unknown>,
    successMessage: string,
  ) => {
    setControlError(null);
    try {
      await action();
      NotificationHandler.instance.success(successMessage);
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Simulation action failed";
      setControlError(message);
      NotificationHandler.instance.error(message);
    }
  };

  const start = async (values: Record<string, FormValue>) => {
    const speed = Number(values.speed);
    const seed = typeof values.seed === "string" ? values.seed.trim() : "";

    return runControl(
      async () => {
        const gatewayBridge = await profileConfiguration.getGatewayBridge();
        await simulation.start({
          speed: gatewayBridge.enabled || gateways.some((gateway) => gateway.type === GatewayType.Real) ? 1 : speed,
          ...(seed ? { seed: Number(seed) } : {}),
          gatewayBridge,
        });
      },
      "Simulation started",
    );
  };

  const updateSpeed = useCallback(
    async (speed: number) => {
      try {
        await simulation.setSpeed({ speed });
        setControlError(null);
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "Unable to update simulation speed";
        setControlError(message);
        NotificationHandler.instance.error(message);
      }
    },
    [simulation],
  );

  const controlFields = useMemo(
    () => [
      rangeField({
        name: "speed",
        label: "Speed",
        value: "1",
        error: null,
        min: 0.1,
        max: 10,
        step: 0.1,
        disabled: requiresRealtime,
        formatValue: (value: string) => `${Number(value).toFixed(1)}x`,
        onChange: (logic) => {
          const value = Number(logic.fieldsState.speed.value);

          if ((isRunning || isPaused) && Number.isFinite(value) && value > 0) {
            void updateSpeed(value);
          }
        },
      }),
      textField({
        name: "seed",
        label: "Seed (optional)",
        value: "",
        error: null,
        placeholder: "Optional",
        disabled: !canStart,
        format: (raw: string) => raw.replace(/[^0-9]/g, ""),
      }),
    ],
    [canStart, isPaused, isRunning, requiresRealtime, updateSpeed],
  );

  return (
    <main className="simulation-dashboard page">
      <header className="simulation-dashboard__header">
        <div>
          <p className="simulation-dashboard__eyebrow">
            LWN SIMULATOR / REALTIME ENGINE
          </p>
          <h1>Simulation control center</h1>
          <p className="simulation-dashboard__subtitle">
            Monitor propagation, protocol traffic and gateway connectivity in
            real time.
          </p>
        </div>
        <div className="simulation-dashboard__status">
          <span className={`simulation-status ${isRunning ? "is-live" : ""}`}>
            <span /> {formatEventType(status)}
          </span>
          <span className="simulation-dashboard__connection">
            WS {simulation.connectionState}
          </span>
        </div>
      </header>

      <SimulationDashboardControls
        fields={controlFields}
        status={status}
        canStart={canStart}
        requiresRealtime={requiresRealtime}
        hardwareLoaded={hardwareLoaded}
        hasActiveHardware={hasActiveHardware}
        activeSimulationIsElsewhere={activeSimulationIsElsewhere}
        activeSimulationProfileName={activeSimulationProfile?.name}
        onStart={start}
        onPause={() => runControl(simulation.pause, "Simulation paused")}
        onResume={() => runControl(simulation.resume, "Simulation resumed")}
        onStop={() => runControl(simulation.stop, "Simulation stopped")}
      />

      {(controlError || simulation.error) && (
        <div className="simulation-dashboard__error" role="alert">
          {controlError ?? simulation.error?.message}
        </div>
      )}

      <SimulationDashboardMetrics snapshot={snapshot} recentRates={recentRates} />

      <SimulationCommandPanel
        devices={devices}
        enabled={isRunning || isPaused}
        onQueueUplink={simulation.queueUplink}
        onQueueDownlink={simulation.queueDownlink}
        onQueueMACCommand={simulation.queueMACCommand}
      />

      <SimulationDashboardTopology
        devices={devices}
        gateways={gateways}
        mapLogic={mapLogic}
        links={mapLinks}
        events={simulation.events}
      />

      <SimulationDashboardRuntime
        devices={snapshot?.devices ?? []}
        deviceProtocols={deviceProtocols}
        gateways={snapshot?.gateways ?? []}
        formatState={formatEventType}
      />

      {isLoadingSnapshot && (
        <div className="simulation-dashboard__loading">
          <Spinner /> Loading simulation state…
        </div>
      )}
    </main>
  );
};

export default SimulationDashboardPage;
