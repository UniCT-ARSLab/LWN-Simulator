"use client";

import { DeviceRegion } from "@lwn-simulator/contracts";
import { useDeviceService } from "@lwn-simulator/sdk";
import {
  Button,
  NotificationHandler,
  PageHeader,
  Table,
  TableRecordItemSort,
} from "@lwn-simulator/ui-components";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

const DevicesPage = () => {
  const deviceService = useDeviceService();
  const router = useRouter();

  useEffect(() => {
    if (deviceService.error) {
      NotificationHandler.instance.error("Failed to load devices");
    }
  }, [deviceService.error]);

  return (
    <div className="page devices-page">
      <PageHeader title="Devices" subTitle="Create and manage devices">
        <Button value="Add Device" onClick={() => router.push("/hardware/devices/new")} />
      </PageHeader>
      <Table
        rowLabels={[
          { value: "name", label: "Name", sort: TableRecordItemSort.Alphabetic },
          { value: "devEUI", label: "DevEUI", sort: TableRecordItemSort.Alphabetic },
          { value: "status", label: "Status" },
          { value: "region", label: "Region" },
        ]}
        orderBy="name"
        filters={[
          { field: "name", label: "Name", placeholder: "Search by name" },
          { field: "devEUI", label: "DevEUI", placeholder: "Search by DevEUI" },
          {
            field: "status",
            label: "Status",
            type: "select",
            options: [
              { label: "Active", value: "active" },
              { label: "Inactive", value: "inactive" },
            ],
          },
          {
            field: "region",
            label: "Region",
            type: "select",
            options: Object.values(DeviceRegion).map((region) => ({ label: region, value: region })),
          },
        ]}
        records={
          Array.isArray(deviceService.devices)
            ? deviceService.devices.map((device) => ({
                id: device.id,
                items: [
                  { label: "name", value: device.name },
                  { label: "devEUI", value: device.devEUI },
                  { label: "status", value: device.active ? "active" : "inactive" },
                  { label: "region", value: device.locationConfig.region },
                ],
              }))
            : []
        }
        pageSize={6}
        onRowClick={(row) => {
          if (row.id) {
            router.push(`/hardware/devices/device-info?deviceId=${row.id}`);
          }
        }}
        isLoading={!Array.isArray(deviceService.devices)}
      />
    </div>
  );
};

export default DevicesPage;
