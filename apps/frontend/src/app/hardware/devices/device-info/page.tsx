"use client";

import { Device, RadioProtocol, RadioReceiveMode } from "@lwn-simulator/contracts";
import { useDeviceService } from "@lwn-simulator/sdk";
import {
  Button,
  EntityDetailSection,
  EntityDetailTone,
  EntityDetails,
  EntityDetailsSectionLayout,
  EntityDetailValueFormat,
  PageHeader,
  NotificationHandler,
  Spinner,
} from "@lwn-simulator/ui-components";
import { DeleteHardwareButton } from "@/components/Hardware";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

const displayValue = (value: unknown) =>
  value === undefined || value === null || value === ""
    ? "Not configured"
    : String(value);

const displayBoolean = (value: boolean) => (value ? "Enabled" : "Disabled");

const getDeviceSections = (device: Device): Array<EntityDetailSection> => {
  const provisioningItems = device.OOTAConfig
    ? [
        { label: "JoinEUI", value: device.OOTAConfig.joinEUI, mono: true },
        { label: "AppKey", value: device.OOTAConfig.appKey, mono: true },
      ]
    : device.ABPConfig
      ? [
          { label: "DevAddr", value: device.ABPConfig.devAddr, mono: true },
          { label: "NwkSKey", value: device.ABPConfig.nwkSKey, mono: true },
          { label: "AppSKey", value: device.ABPConfig.appSKey, mono: true },
        ]
      : [{ label: "Credentials", value: "Not configured" }];

  return [
    {
      title: "Device overview",
      icon: "memory",
      description: "Identity, state and radio operating profile",
      layout: EntityDetailsSectionLayout.Full,
      metrics: [
        {
          label: "Status",
          value: device.active ? "Active" : "Inactive",
          tone: device.active
            ? EntityDetailTone.Positive
            : EntityDetailTone.Muted,
          icon: device.active ? "check_circle" : "pause_circle",
        },
        {
          label: "Class",
          value: device.class,
          format: EntityDetailValueFormat.Enum,
        },
        {
          label: "Activation",
          value: device.activation,
          format: EntityDetailValueFormat.Enum,
        },
        {
          label: "Region",
          value: device.locationConfig.region,
          format: EntityDetailValueFormat.Enum,
        },
      ],
      items: [
        { label: "Name", value: device.name },
        { label: "Device ID", value: device.id, mono: true },
        { label: "DevEUI", value: device.devEUI, mono: true },
        { label: "Radio Protocol", value: device.radioProtocol === RadioProtocol.RawLora ? "Raw LoRa" : "LoRaWAN" },
        { label: "Radio Receive Mode", value: device.radioReceiveMode === RadioReceiveMode.Auto ? "Auto" : device.radioReceiveMode === RadioReceiveMode.RawLora ? "Raw LoRa" : "LoRaWAN" },
      ],
    },
    {
      title: "Location",
      icon: "location_on",
      description: "Physical position used by the simulation",
      metrics: [
        { label: "Latitude", value: device.locationConfig.latitude },
        { label: "Longitude", value: device.locationConfig.longitude },
        { label: "Altitude", value: `${device.locationConfig.altitude} m` },
      ],
      items: [
        {
          label: "Region",
          value: device.locationConfig.region,
          format: EntityDetailValueFormat.Enum,
        },
      ],
    },
    {
      title: device.OOTAConfig ? "OTAA provisioning" : "ABP provisioning",
      icon: "key",
      description: device.OOTAConfig
        ? "Credentials used to join the LoRaWAN network"
        : "Preconfigured session credentials",
      items: provisioningItems,
    },
    {
      title: "Radio configuration",
      icon: "settings_input_antenna",
      description: "Receive windows and downlink timing parameters",
      layout: EntityDetailsSectionLayout.Full,
      metrics: [
        { label: "RX1 delay", value: device.RX1Config.delay },
        { label: "RX2 delay", value: device.RX2Config.delay },
        {
          label: "Channel frequency",
          value: device.RX2Config.channelFrequency,
        },
        { label: "ACK timeout", value: device.RX2Config.ACKTimeout },
      ],
      items: [
        { label: "RX1 duration", value: device.RX1Config.duration },
        {
          label: "RX1 data rate offset",
          value: device.RX1Config.dataRateOffset,
        },
        { label: "RX2 duration", value: device.RX2Config.duration },
        { label: "RX2 data rate", value: device.RX2Config.dataRate },
      ],
    },
    {
      title: "Frame and payload",
      icon: "data_object",
      description: "Uplink framing, counters and application payload",
      layout: EntityDetailsSectionLayout.Full,
      metrics: [
        { label: "FPort", value: device.frameConfig.fPort },
        {
          label: "Uplink interval",
          value: device.payloadConfig.uplinkInterval,
        },
        {
          label: "Message type",
          value: device.payloadConfig.MType,
          format: EntityDetailValueFormat.Enum,
        },
        {
          label: "Counter validation",
          value: device.frameConfig.disableFrameCounterValidation
            ? "Disabled"
            : "Enabled",
          tone: device.frameConfig.disableFrameCounterValidation
            ? EntityDetailTone.Warning
            : EntityDetailTone.Positive,
        },
      ],
      items: [
        { label: "Retransmission", value: device.frameConfig.retransmission },
        { label: "FCnt up", value: displayValue(device.frameConfig.FCntUp) },
        {
          label: "FCnt down",
          value: displayValue(device.frameConfig.FCntDown),
        },
        {
          label: "Oversized payload",
          value: device.payloadConfig.oversizedPayloadBehavior,
          format: EntityDetailValueFormat.Enum,
        },
        { label: "Payload", value: device.payloadConfig.payload, mono: true },
        {
          label: "Base64 encoded",
          value: displayBoolean(device.payloadConfig.base64Encoded),
        },
        {
          label: "Payload Codec",
          value: displayBoolean(device.payloadConfig.payloadCodec?.enabled ?? false),
        },
        {
          label: "JavaScript Payload Codec",
          value: displayValue(device.payloadConfig.payloadCodec?.script),
          mono: true,
        },
      ],
    },
    {
      title: "Advanced configuration",
      icon: "tune",
      description: "Adaptive data rate and antenna simulation parameters",
      metrics: [
        { label: "Antenna range", value: device.advancedConfig.antennaRange },
        {
          label: "Adaptive data rate",
          value: displayBoolean(device.advancedConfig.ADREnabled),
          tone: device.advancedConfig.ADREnabled
            ? EntityDetailTone.Positive
            : EntityDetailTone.Muted,
        },
      ],
      items: [],
    },
    {
      title: "Simulation behavior",
      icon: "monitoring",
      description: "Runtime telemetry and packet activity will appear here",
      emptyState: {
        icon: "query_stats",
        title: "Simulation data not available yet",
        description:
          "This panel is reserved for live telemetry, packet history and behavior metrics.",
      },
      items: [],
    },
  ];
};

const DeviceInfoContent = () => {
  const deviceService = useDeviceService();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [device, setDevice] = useState<Device | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const deviceId = searchParams.get("deviceId");

    if (!deviceId) {
      setError(new Error("Device identifier is missing"));
      setIsLoading(false);
      return;
    }

    let isCurrent = true;

    const loadDevice = async () => {
      try {
        const result = await deviceService.getDevice({ id: deviceId });

        if (isCurrent) {
          setDevice(result);
        }
      } catch (err) {
        if (isCurrent) {
          NotificationHandler.instance.error("Failed to load device");
          setError(
            err instanceof Error ? err : new Error("Failed to load device"),
          );
        }
      } finally {
        if (isCurrent) {
          setIsLoading(false);
        }
      }
    };

    loadDevice();

    return () => {
      isCurrent = false;
    };
  }, [deviceService, searchParams]);

  if (isLoading) {
    return (
      <div className="page entity-details-loading">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="page device-page">
      <PageHeader
        title={device?.name || "Device details"}
        subTitle="View device configuration and status"
      >
        <Button value="Back" onClick={() => router.back()} />
        {device && <Button value="Edit" onClick={() => router.push("/hardware/devices/new?deviceId=" + encodeURIComponent(device.id))} />}
        {device && (
          <DeleteHardwareButton
            key={device.id}
            kind="device"
            name={device.name}
            onDelete={async () => {
              await deviceService.deleteDevice({ id: device.id });
            }}
            onDeleted={() => {
              NotificationHandler.instance.success("Device deleted");
              router.back();
            }}
          />
        )}
      </PageHeader>
      {error || !device ? (
        <p>Unable to load this device.</p>
      ) : (
        <EntityDetails sections={getDeviceSections(device)} />
      )}
    </div>
  );
};

const DevicePage = () => {
  return (
    <Suspense
      fallback={
        <div className="page entity-details-loading">
          <Spinner />
        </div>
      }
    >
      <DeviceInfoContent />
    </Suspense>
  );
};

export default DevicePage;
