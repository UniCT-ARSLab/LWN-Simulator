"use client";

import { memo } from "react";
import { RadioProtocol } from "@lwn-simulator/contracts";
import { SimulationRuntimeList } from "../SimulationRuntimeList";
import { DeviceRuntimeListProps } from "./DeviceRuntimeList.types";

export const DeviceRuntimeList = memo((props: DeviceRuntimeListProps) => {
  return (
    <SimulationRuntimeList
      items={props.devices}
      emptyMessage="No runtime devices."
      renderItem={(device) => (
        <>
          <span>{device.name}</span>
          <strong>
            {!device.active
              ? "Inactive"
              : props.deviceProtocols?.[device.id] === RadioProtocol.RawLora
                ? "Raw LoRa · No join required"
                : `${device.joined ? "Joined" : "Joining"} · FCnt ${device.frameCounterUp}`}
          </strong>
        </>
      )}
    />
  );
});

DeviceRuntimeList.displayName = "DeviceRuntimeList";
