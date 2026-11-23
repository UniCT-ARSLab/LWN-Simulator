"use client";

import { GatewayForm } from "@/components/Hardware";
import {
  CreateGatewayRequest,
  Gateway,
  GatewayType,
} from "@lwn-simulator/contracts";
import { useGatewayService, useProfileService } from "@lwn-simulator/sdk";
import {
  FormValue,
  NotificationHandler,
  Spinner,
  PageHeader,
} from "@lwn-simulator/ui-components";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

const NewGatewayPage = () => {
  // Hooks
  const gatewayService = useGatewayService();
  const profileService = useProfileService();
  const router = useRouter();
  const searchParams = useSearchParams();
  const gatewayId = searchParams.get("gatewayId");
  const simulationActive = profileService.simulationActivity?.active === true &&
    (!profileService.simulationActivity.profileID ||
      profileService.simulationActivity.profileID === profileService.activeProfileID);

  // States
  const [editingGateway, setEditingGateway] = useState<Gateway | null>(null);
  const [isLoadingEdit, setIsLoadingEdit] = useState(!!gatewayId);
  const editLocked = editingGateway !== null && simulationActive;

  // Effects
  useEffect(() => {
    if (!gatewayId) {
      setEditingGateway(null);
      setIsLoadingEdit(false);
      return;
    }
    let isCurrent = true;
    setIsLoadingEdit(true);
    gatewayService.getGateway({ id: gatewayId }).then((gateway) => {
      if (isCurrent) setEditingGateway(gateway);
    }).catch(() => {
      if (isCurrent) NotificationHandler.instance.error("Failed to load gateway");
    }).finally(() => {
      if (isCurrent) setIsLoadingEdit(false);
    });
    return () => { isCurrent = false; };
  }, [gatewayId, gatewayService]);

  // Callbacks
  const handleSubmit = async (values: Record<string, FormValue>) => {
    if (editingGateway && simulationActive) {
      try {
        await gatewayService.updateGateway({
          id: editingGateway.id,
          gateway: {
            ...editingGateway,
            active: values.active as boolean,
            latitude: Number(values.latitude),
            longitude: Number(values.longitude),
            altitude: Number(values.altitude),
          },
        });
        NotificationHandler.instance.success("Gateway updated");
        router.back();
      } catch {
        NotificationHandler.instance.error("Failed to update gateway");
      }
      return;
    }

    const type = values.type as GatewayType;
    const req: CreateGatewayRequest = {
      name: values.name as string,
      type,
      macAddress: values.macAddress as string,
      gatewayEUI: values.gatewayEUI as string,
      latitude: Number(values.latitude),
      longitude: Number(values.longitude),
      altitude: Number(values.altitude),
      ...(type === GatewayType.Virtual
        ? { keepAlive: Number(values.keepAlive) }
        : {
            gatewayIPv4: values.gatewayIPv4 as string,
            gatewayPort: Number(values.gatewayPort),
          }),
    };

    try {
      if (editingGateway) {
        await gatewayService.updateGateway({
          id: editingGateway.id,
          gateway: { ...editingGateway, ...req, type: editingGateway.type, macAddress: editingGateway.macAddress, gatewayEUI: editingGateway.gatewayEUI, active: values.active as boolean },
        });
        NotificationHandler.instance.success("Gateway updated");
        router.back();
      } else {
        await gatewayService.createGateway(req);
        NotificationHandler.instance.success("Gateway created");
        router.push("/hardware/gateways");
      }
    } catch {
      NotificationHandler.instance.error(editingGateway ? "Failed to update gateway" : "Failed to create gateway");
    }
  };

  if (isLoadingEdit) return <div className="page entity-details-loading"><Spinner /></div>;

  return (
    <div className="page new-device-page new-gateway-page">
      <PageHeader
        title={editingGateway ? "Edit Gateway" : "Create new Gateway"}
        subTitle={
          editLocked
            ? "Simulation is active. You can change the gateway status and position."
            : undefined
        }
      />
      <GatewayForm
        gateway={editingGateway}
        simulationActive={simulationActive}
        onSubmit={handleSubmit}
      />
    </div>
  );
};

const NewGatewayPageWithSuspense = () => (
  <Suspense fallback={<div className="page entity-details-loading"><Spinner /></div>}>
    <NewGatewayPage />
  </Suspense>
);

export default NewGatewayPageWithSuspense;
