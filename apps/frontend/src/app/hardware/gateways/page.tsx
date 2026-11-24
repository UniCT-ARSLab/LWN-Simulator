"use client";

import { useGatewayService } from "@lwn-simulator/sdk";
import {
  Button,
  ButtonType,
  NotificationHandler,
  PageHeader,
  Table,
  TableRecordItemSort,
} from "@lwn-simulator/ui-components";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

const GatewaysPage = () => {
  const gatewayService = useGatewayService();
  const router = useRouter();

  useEffect(() => {
    if (gatewayService.error) {
      NotificationHandler.instance.error("Failed to load gateways");
    }
  }, [gatewayService.error]);

  return (
    <div className="page gateways-page">
      <PageHeader title="Gateways" subTitle="Create and manage virtual and real gateways">
        <Button
          value="Gateway Bridge"
          type={ButtonType.Outlined}
          onClick={() => router.push("/hardware/gateway-bridge")}
        />
        <Button value="Add Gateway" onClick={() => router.push("/hardware/gateways/new")} />
      </PageHeader>
      <Table
        rowLabels={[
          { value: "name", label: "Name", sort: TableRecordItemSort.Alphabetic },
          { value: "type", label: "Type", sort: TableRecordItemSort.Alphabetic },
          { value: "status", label: "Status" },
          { value: "location", label: "Location" },
          { value: "gatewayEUI", label: "Gateway EUI", sort: TableRecordItemSort.Alphabetic },
        ]}
        orderBy="name"
        filters={[
          { field: "name", label: "Name", placeholder: "Search by name" },
          { field: "gatewayEUI", label: "Gateway EUI", placeholder: "Search by EUI" },
          {
            field: "type",
            label: "Type",
            type: "select",
            options: [
              { label: "Virtual", value: "virtual" },
              { label: "Real", value: "real" },
            ],
          },
          {
            field: "status",
            label: "Status",
            type: "select",
            options: [
              { label: "Active", value: "active" },
              { label: "Inactive", value: "inactive" },
            ],
          },
        ]}
        records={
          Array.isArray(gatewayService.gateways)
            ? gatewayService.gateways.map((gateway) => ({
                id: gateway.id,
                items: [
                  { label: "name", value: gateway.name },
                  { label: "type", value: gateway.type },
                  { label: "status", value: gateway.active ? "active" : "inactive" },
                  { label: "location", value: `${gateway.latitude}, ${gateway.longitude}` },
                  { label: "gatewayEUI", value: gateway.gatewayEUI },
                ],
              }))
            : []
        }
        pageSize={6}
        onRowClick={(row) => {
          if (row.id) {
            router.push(`/hardware/gateways/gateway-info?gatewayId=${row.id}`);
          }
        }}
        isLoading={!Array.isArray(gatewayService.gateways)}
      />
    </div>
  );
};

export default GatewaysPage;
