"use client";

import { Gateway, GatewayType } from "@lwn-simulator/contracts";
import { useGatewayService } from "@lwn-simulator/sdk";
import {
  Button,
  EntityDetailTone,
  EntityDetailSection,
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

const getGatewaySections = (gateway: Gateway): Array<EntityDetailSection> => {
  return [
    {
      title: "Gateway overview",
      icon: "router",
      description: "Identity, state and network operating profile",
      layout: EntityDetailsSectionLayout.Full,
      metrics: [
        {
          label: "Status",
          value: gateway.active ? "Active" : "Inactive",
          tone: gateway.active
            ? EntityDetailTone.Positive
            : EntityDetailTone.Muted,
          icon: gateway.active ? "check_circle" : "pause_circle",
        },
        {
          label: "Type",
          value: gateway.type,
          format: EntityDetailValueFormat.Enum,
        },
        gateway.type === GatewayType.Virtual
          ? {
              label: "Keep alive",
              value: `${displayValue(gateway.keepAlive)} s`,
              icon: "schedule",
            }
          : {
              label: "Gateway port",
              value: displayValue(gateway.gatewayPort),
              icon: "lan",
            },
        {
          label: "Location",
          value: `${gateway.latitude}, ${gateway.longitude}`,
          mono: true,
          icon: "location_on",
        },
      ],
      items: [
        { label: "Name", value: gateway.name },
        { label: "Gateway ID", value: gateway.id, mono: true },
        { label: "MAC address", value: gateway.macAddress, mono: true },
        { label: "Gateway EUI", value: gateway.gatewayEUI, mono: true },
      ],
    },
    {
      title: "Connection",
      icon: "lan",
      description:
        gateway.type === GatewayType.Virtual
          ? "Virtual gateway heartbeat configuration"
          : "Network endpoint of the physical gateway",
      metrics:
        gateway.type === GatewayType.Virtual
          ? [
              {
                label: "Keep alive",
                value: `${displayValue(gateway.keepAlive)} seconds`,
                icon: "schedule",
              },
            ]
          : [
              {
                label: "IPv4 address",
                value: displayValue(gateway.gatewayIPv4),
                mono: true,
              },
              {
                label: "Port",
                value: displayValue(gateway.gatewayPort),
                icon: "lan",
              },
            ],
      items:
        gateway.type === GatewayType.Real
          ? [{ label: "Protocol", value: "IPv4 endpoint" }]
          : [{ label: "Protocol", value: "Virtual simulation link" }],
    },
    {
      title: "Location",
      icon: "location_on",
      description: "Physical position used by the simulation",
      metrics: [
        { label: "Latitude", value: gateway.latitude },
        { label: "Longitude", value: gateway.longitude },
        { label: "Altitude", value: `${gateway.altitude} m` },
      ],
      items: [
        { label: "Coordinates", value: `${gateway.latitude}, ${gateway.longitude}`, mono: true },
      ],
    },
    {
      title: "Simulation behavior",
      icon: "monitoring",
      description: "Traffic, forwarding and routing data will appear here",
      layout: EntityDetailsSectionLayout.Full,
      emptyState: {
        icon: "query_stats",
        title: "Simulation data not available yet",
        description:
          "This panel is reserved for gateway traffic, packet forwarding, channel performance and routed nodes.",
      },
      items: [],
    },
  ];
};

const GatewayInfoContent = () => {
  const gatewayService = useGatewayService();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [gateway, setGateway] = useState<Gateway | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const gatewayId = searchParams.get("gatewayId");

    if (!gatewayId) {
      setError(new Error("Gateway identifier is missing"));
      setIsLoading(false);
      return;
    }

    let isCurrent = true;

    const loadGateway = async () => {
      try {
        const result = await gatewayService.getGateway({ id: gatewayId });

        if (isCurrent) {
          setGateway(result);
        }
      } catch (err) {
        if (isCurrent) {
          NotificationHandler.instance.error("Failed to load gateway");
          setError(
            err instanceof Error ? err : new Error("Failed to load gateway"),
          );
        }
      } finally {
        if (isCurrent) {
          setIsLoading(false);
        }
      }
    };

    loadGateway();

    return () => {
      isCurrent = false;
    };
  }, [gatewayService, searchParams]);

  if (isLoading) {
    return (
      <div className="page entity-details-loading">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="page gateway-page">
      <PageHeader
        title={gateway?.name || "Gateway details"}
        subTitle="View gateway configuration and status"
      >
        <Button
          value="Back"
          onClick={() => router.back()}
        />
        {gateway && (
          <Button
            value="Edit"
            onClick={() => router.push("/hardware/gateways/new?gatewayId=" + encodeURIComponent(gateway.id))}
          />
        )}
        {gateway && (
          <DeleteHardwareButton
            key={gateway.id}
            kind="gateway"
            name={gateway.name}
            onDelete={async () => {
              await gatewayService.deleteGateway({ id: gateway.id });
            }}
            onDeleted={() => {
              NotificationHandler.instance.success("Gateway deleted");
              router.back();
            }}
          />
        )}
      </PageHeader>
      {error || !gateway ? (
        <p>Unable to load this gateway.</p>
      ) : (
        <EntityDetails sections={getGatewaySections(gateway)} />
      )}
    </div>
  );
};

const GatewayInfoPage = () => (
  <Suspense
    fallback={
      <div className="page entity-details-loading">
        <Spinner />
      </div>
    }
  >
    <GatewayInfoContent />
  </Suspense>
);

export default GatewayInfoPage;
