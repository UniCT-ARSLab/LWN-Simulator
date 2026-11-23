"use client";

import { SensorMap, SensorMapMode, useSensorMap } from "@/components/SensorMap";
import {
  getAltitude,
  latValidation,
  lngValidation,
  randomGatewayEUI,
  randomMacAddress,
} from "@/utils";
import { Gateway, GatewayType } from "@lwn-simulator/contracts";
import {
  booleanCheckboxField,
  Button,
  ButtonLayout,
  ButtonType,
  Form,
  FormField,
  FormLogic,
  FormValue,
  NotificationHandler,
  radioField,
  textField,
} from "@lwn-simulator/ui-components";
import { useCallback, useEffect, useRef } from "react";
import { GatewayFormProps } from "./GatewayForm.types";

const lockFieldsDuringSimulation = (
  fields: Array<FormField>,
  locked: boolean,
): Array<FormField> =>
  fields.map((field) =>
    locked && !["active", "latitude", "longitude", "altitude"].includes(field.name)
      ? { ...field, disabled: true, toolbar: undefined }
      : field,
  );

export const GatewayForm = (props: GatewayFormProps) => {
  // Hooks
  const formLogicRef = useRef<FormLogic | null>(null);
  const hydratedGatewayIdRef = useRef<string | null>(null);
  const mapLogic = useSensorMap({ mode: SensorMapMode.Coords });
  const { updatePos } = mapLogic;

  // Callbacks
  const hydrateForm = useCallback((logic: FormLogic, gateway: Gateway) => {
    const values: Record<string, FormValue> = {
      active: gateway.active,
      name: gateway.name,
      type: gateway.type,
      macAddress: gateway.macAddress,
      gatewayEUI: gateway.gatewayEUI,
      keepAlive: gateway.keepAlive?.toString() ?? "",
      gatewayIPv4: gateway.gatewayIPv4 ?? "",
      gatewayPort: gateway.gatewayPort?.toString() ?? "",
      latitude: gateway.latitude.toString(),
      longitude: gateway.longitude.toString(),
      altitude: gateway.altitude.toString(),
    };
    Object.entries(values).forEach(([name, value]) => logic.setValue(name, value));

    updatePos(gateway.latitude, gateway.longitude);
  }, [updatePos]);

  const coordinateFormat = (raw: string) =>
    raw
      .replace(/[^0-9.-]/g, "")
      .replace(/(?!^)-/g, "")
      .replace(/(\..*)\./g, "$1");

  const updateMapPosition = (logic: FormLogic) => {
    if (
      logic.fieldsState.latitude.value &&
      (logic.fieldsState.latitude.value as string).match(latValidation) &&
      logic.fieldsState.longitude.value &&
      (logic.fieldsState.longitude.value as string).match(lngValidation)
    ) {
      mapLogic.updatePos(
        Number(logic.fieldsState.latitude.value),
        Number(logic.fieldsState.longitude.value),
      );
    }
  };

  // Effects
  useEffect(() => {
    if (
      props.gateway &&
      formLogicRef.current &&
      hydratedGatewayIdRef.current !== props.gateway.id
    ) {
      hydrateForm(formLogicRef.current, props.gateway);
      hydratedGatewayIdRef.current = props.gateway.id;
    }
  }, [hydrateForm, props.gateway]);

  useEffect(() => {
    if (formLogicRef.current && mapLogic.selectedPos) {
      const { latitude, longitude } = formLogicRef.current.fieldsState;
      if (latitude.value !== mapLogic.selectedPos.lat.toString()) {
        formLogicRef.current.setValue(
          "latitude",
          mapLogic.selectedPos.lat.toString(),
        );
      }
      if (longitude.value !== mapLogic.selectedPos.lng.toString()) {
        formLogicRef.current.setValue(
          "longitude",
          mapLogic.selectedPos.lng.toString(),
        );
      }
    }
  }, [mapLogic.selectedPos]);

  return (
    <div className="page-content gateway-form">
      <div className="form-wrapper">
        <Form
          onSubmit={props.onSubmit}
          submitButton={{
            value: props.gateway ? "Save" : "Create",
            className: "submit-button",
          }}
          onLogicReady={(logic) => {
            formLogicRef.current = logic;
            if (
              props.gateway &&
              hydratedGatewayIdRef.current !== props.gateway.id
            ) {
              hydrateForm(logic, props.gateway);
              hydratedGatewayIdRef.current = props.gateway.id;
            }
          }}
          fields={lockFieldsDuringSimulation([
            booleanCheckboxField({
              value: props.gateway?.active ?? false,
              name: "active",
              label: "Active",
              error: null,
              text: "Enable",
              display: !!props.gateway,
            }),
            textField({
              name: "name",
              label: "Name",
              value: "",
              placeholder: "Gateway Name",
              error: null,
              required: true,
            }),
            radioField({
              value: null,
              name: "type",
              label: "Gateway Type",
              error: null,
              options: Object.values(GatewayType).map((value) => ({
                value,
                displayed: <>{value.toUpperCase()}</>,
              })),
              info: {
                default: "Defines how the gateway connects to the simulator",
                [GatewayType.Virtual]:
                  "Software gateway simulated locally with a keep-alive interval",
                [GatewayType.Real]:
                  "Physical gateway reachable through an IPv4 address and port",
              },
              required: true,
              disabled: !!props.gateway,
            }),
            textField({
              name: "macAddress",
              label: "MAC Address",
              value: "",
              placeholder: "02:00:00:00:00:01",
              format: (raw: string) =>
                raw
                  .replace(/[^0-9A-Fa-f:]/g, "")
                  .slice(0, 17)
                  .toUpperCase(),
              info: {
                default: "Unique hardware network address of the gateway interface",
              },
              validations: [
                {
                  rule: /^(?:[0-9A-F]{2}:){5}[0-9A-F]{2}$/,
                  error: "Invalid MAC Address",
                },
              ],
              error: null,
              required: true,
              disabled: !!props.gateway,
              toolbar: props.gateway ? undefined : (
                <Button
                  value={<span className="material-symbols-outlined">cached</span>}
                  layout={ButtonLayout.Icon}
                  type={ButtonType.Outlined}
                  onClick={() =>
                    formLogicRef.current?.setValue(
                      "macAddress",
                      randomMacAddress(),
                    )
                  }
                />
              ),
            }),
            textField({
              name: "gatewayEUI",
              label: "Gateway EUI",
              value: "",
              placeholder: "0102030405060708",
              format: (raw: string) =>
                raw
                  .replace(/[^0-9A-Fa-f]/g, "")
                  .slice(0, 16)
                  .toUpperCase(),
              info: {
                default:
                  "Unique identifier assigned to the gateway in the LoRaWAN network",
              },
              validations: [
                {
                  rule: /^[0-9A-F]{16}$/,
                  error: "Invalid Gateway EUI",
                },
              ],
              error: null,
              required: true,
              disabled: !!props.gateway,
              toolbar: props.gateway ? undefined : (
                <Button
                  value={<span className="material-symbols-outlined">cached</span>}
                  layout={ButtonLayout.Icon}
                  type={ButtonType.Outlined}
                  onClick={() =>
                    formLogicRef.current?.setValue(
                      "gatewayEUI",
                      randomGatewayEUI(),
                    )
                  }
                />
              ),
            }),
            textField({
              name: "keepAlive",
              label: "Keep Alive (seconds)",
              value: "",
              placeholder: "30",
              format: (raw: string) => raw.replace(/[^0-9]/g, ""),
              info: {
                default:
                  "Interval, expressed in seconds, used by a virtual gateway to stay connected",
              },
              validations: [
                {
                  rule: /^[1-9][0-9]*$/,
                  error: "Keep Alive must be greater than 0 seconds",
                },
              ],
              error: null,
              required: true,
              display: (fieldsState: Record<string, FormField>) =>
                fieldsState.type.value === GatewayType.Virtual,
            }),
            textField({
              name: "gatewayIPv4",
              label: "Gateway IPv4",
              value: "",
              placeholder: "192.168.1.10",
              info: {
                default: "IPv4 address where the real gateway can be reached",
              },
              validations: [
                {
                  rule:
                    /^(?:(?:25[0-5]|2[0-4][0-9]|1?[0-9]?[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1?[0-9]?[0-9])$/,
                  error: "Invalid IPv4 Address",
                },
              ],
              error: null,
              required: true,
              display: (fieldsState: Record<string, FormField>) =>
                fieldsState.type.value === GatewayType.Real,
            }),
            textField({
              name: "gatewayPort",
              label: "Gateway Port",
              value: "",
              placeholder: "1700",
              format: (raw: string) => raw.replace(/[^0-9]/g, "").slice(0, 5),
              info: { default: "Network port exposed by the real gateway" },
              validations: [
                {
                  rule:
                    /^(?:[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$/,
                  error: "Invalid Gateway Port",
                },
              ],
              error: null,
              required: true,
              display: (fieldsState: Record<string, FormField>) =>
                fieldsState.type.value === GatewayType.Real,
            }),
            textField({
              name: "latitude",
              label: "Latitude",
              placeholder: "45.4642",
              value: "",
              error: null,
              info: { default: "Decimal degrees (-90 to 90)" },
              format: coordinateFormat,
              validations: [{ rule: latValidation, error: "Invalid Latitude" }],
              required: true,
              onChange: updateMapPosition,
            }),
            textField({
              name: "longitude",
              label: "Longitude",
              placeholder: "9.1900",
              value: "",
              error: null,
              info: { default: "Decimal degrees (-180 to 180)" },
              format: coordinateFormat,
              validations: [{ rule: lngValidation, error: "Invalid Longitude" }],
              required: true,
              onChange: updateMapPosition,
            }),
            textField({
              name: "altitude",
              label: "Altitude",
              placeholder: "120",
              value: "",
              error: null,
              info: { default: "Meters above sea level" },
              format: coordinateFormat,
              validations: [
                {
                  rule: /^-?\d+(?:\.\d+)?$/,
                  error: "Invalid Altitude",
                },
              ],
              required: true,
              toolbar: (
                <Button
                  value={<span className="material-symbols-outlined">cached</span>}
                  layout={ButtonLayout.Icon}
                  type={ButtonType.Outlined}
                  onClick={async () => {
                    const logic = formLogicRef.current;
                    if (
                      !logic ||
                      !(logic.fieldsState.longitude.value as string).match(
                        lngValidation,
                      ) ||
                      !(logic.fieldsState.latitude.value as string).match(
                        latValidation,
                      )
                    ) {
                      NotificationHandler.instance.error(
                        "Select valid latitude and longitude",
                      );
                      return;
                    }
                    const altitude = await getAltitude(
                      Number(logic.fieldsState.latitude.value),
                      Number(logic.fieldsState.longitude.value),
                    );
                    logic.setValue("altitude", altitude.toString());
                  }}
                />
              ),
            }),
          ], !!props.gateway && props.simulationActive)}
        />
      </div>
      <SensorMap logic={mapLogic} mode={SensorMapMode.Coords} />
    </div>
  );
};
