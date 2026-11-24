import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  DeviceServiceContent,
  DeviceServiceProviderProps,
} from "./DeviceService.types";
import DeviceServiceContext from "./DeviceServiceContext";
import {
  CreateDeviceRequest,
  DeleteDeviceRequest,
  Device,
  GetDeviceRequest,
  RealtimeWebSocketMessage,
  UpdateDeviceRequest,
} from "@lwn-simulator/contracts";
import { DeviceService } from "./DeviceService";
import { useWebSocketService } from "../WebSocketService";
import { RealtimeWebSocketMessageType } from "../../models/RealtimeWebSocket";

const DeviceServiceProvider = (props: DeviceServiceProviderProps) => {
  // States
  const [devices, setDevices] = useState<Array<Device> | null>(null);
  const [error, setError] = useState<Error | null>(null);

  // Hooks
  const { connectionState, subscribe } = useWebSocketService();
  const hasConnectedRef = useRef(false);

  // Memos
  const service = useMemo(
    () => new DeviceService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const createDevice = useCallback(
    async (req: CreateDeviceRequest): Promise<Device> => {
      const res = await service.createDevice(req);

      setError(null);

      setDevices((prev) => {
        if (!prev) return [res];
        const existingIndex = prev.findIndex((device) => device.id === res.id);
        if (existingIndex === -1) return [...prev, res];

        return prev.map((device, index) =>
          index === existingIndex ? res : device,
        );
      });

      return res;
    },
    [service],
  );

  const getDevice = useCallback(
    async (req: GetDeviceRequest): Promise<Device> => {
      return service.getDevice(req);
    },
    [service],
  );

  const getDevices = useCallback(async (): Promise<Array<Device>> => {
    const res = await service.getDevices();

    setDevices(res);
    setError(null);

    return res;
  }, [service]);

  const updateDevice = useCallback(
    async (req: UpdateDeviceRequest): Promise<Device> => {
      const res = await service.updateDevice(req);

      setDevices((prev) =>
        prev?.map((device) => (device.id === res.id ? res : device)) ?? null,
      );
      setError(null);

      return res;
    },
    [service],
  );

  const deleteDevice = useCallback(
    async (req: DeleteDeviceRequest): Promise<void> => {
      await service.deleteDevice(req);

      setDevices((prev) =>
        prev?.filter((device) => device.id !== req.id) ?? null,
      );
      setError(null);
    },
    [service],
  );

  const handleRealtimeMessage = useCallback((rawMessage: string) => {
    try {
      const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
      if (message.profileID && message.profileID !== props.profileID) return;
      if (
        (message.type === RealtimeWebSocketMessageType.DeviceCreated ||
          message.type === RealtimeWebSocketMessageType.DeviceUpdated) &&
        message.device
      ) {
        setDevices((previous) => {
          if (!previous) return [message.device as Device];
          const exists = previous.some(
            (device) => device.id === message.device?.id,
          );
          return exists
            ? previous.map((device) =>
                device.id === message.device?.id
                  ? (message.device as Device)
                  : device,
              )
            : [...previous, message.device as Device];
        });
      } else if (
        message.type === RealtimeWebSocketMessageType.DeviceDeleted &&
        message.resourceID
      ) {
        setDevices((previous) =>
          previous?.filter((device) => device.id !== message.resourceID) ?? null,
        );
      }
    } catch {
      return;
    }
  }, [props.profileID]);

  // Memos
  const value = useMemo(
    (): DeviceServiceContent => ({
      createDevice,
      getDevice,
      getDevices,
      updateDevice,
      deleteDevice,
      devices,
      error,
    }),
    [
      createDevice,
      getDevice,
      getDevices,
      updateDevice,
      deleteDevice,
      devices,
      error,
    ],
  );

  // Effects
  useEffect(
    () => subscribe(handleRealtimeMessage),
    [handleRealtimeMessage, props.profileID, subscribe],
  );

  useEffect(() => {
    if (connectionState !== "connected") return;
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true;
      return;
    }

    void getDevices().catch(() => {
      setError(new Error("Failed to recover devices after websocket reconnect"));
    });
  }, [connectionState, getDevices]);

  useEffect(() => {
    (async () => {
      try {
        const res = await getDevices();

        setDevices(res);
      } catch (err) {
        setDevices([]);
        setError(
          err instanceof Error ? err : new Error("Failed to load devices"),
        );
      }
    })();
  }, [getDevices]);

  return (
    <DeviceServiceContext.Provider value={value}>
      {props.children}
    </DeviceServiceContext.Provider>
  );
};

export default DeviceServiceProvider;
