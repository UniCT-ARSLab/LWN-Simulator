"use client";

import { memo } from "react";
import { SimulationRuntimeList } from "../SimulationRuntimeList";
import { GatewayRuntimeListProps } from "./GatewayRuntimeList.types";

export const GatewayRuntimeList = memo((props: GatewayRuntimeListProps) => {
  return (
    <SimulationRuntimeList
      items={props.gateways}
      emptyMessage="No runtime gateways."
      renderItem={(gateway) => (
        <>
          <span>{gateway.name}</span>
          <strong>
            {props.formatState(gateway.gatewayState)} · {gateway.ingressPackets} in / {gateway.egressPackets} out
          </strong>
        </>
      )}
    />
  );
});

GatewayRuntimeList.displayName = "GatewayRuntimeList";
