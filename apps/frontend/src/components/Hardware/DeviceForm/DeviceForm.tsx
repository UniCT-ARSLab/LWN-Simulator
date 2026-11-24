"use client";

import { SensorMap, SensorMapMode, useSensorMap } from "@/components/SensorMap";
import {
  estimateRegion,
  getAltitude,
  hasPayloadDecoder,
  isValidJavaScript,
  latValidation,
  lngValidation,
  randomDevEUI,
  rx1DataRateOffsetRange,
  rx2DataRateOptions,
  rx2Frequency,
} from "@/utils";
import {
  Device,
  DeviceActivation,
  DeviceClass,
  DeviceMType,
  DeviceRegion,
  OversizedPayloadBehavior,
  RadioProtocol,
  RadioReceiveMode,
} from "@lwn-simulator/contracts";
import {
  booleanCheckboxField,
  Button,
  ButtonLayout,
  ButtonType,
  codeEditorField,
  Form,
  FormField,
  FormLogic,
  FormValue,
  NotificationHandler,
  radioField,
  selectField,
  SelectOption,
  textField,
} from "@lwn-simulator/ui-components";
import { useCallback, useEffect, useRef } from "react";
import { DeviceFormProps } from "./DeviceForm.types";

const getRx1DataRateOptions = (region: DeviceRegion): Array<SelectOption> => {
  const [minimum, maximum] = rx1DataRateOffsetRange[region];

  return Array.from({ length: maximum - minimum + 1 }, (_, index) => ({
    value: (minimum + index).toString(),
  }));
};

const getRx2DataRateOptions = (region: DeviceRegion): Array<SelectOption> =>
  rx2DataRateOptions[region].map((option) => ({ value: option }));

const defaultPayloadCodec = `function decodeUplink(input) {
  return { data: {} };
}`;

const lockFieldsDuringSimulation = (
  fields: Array<FormField>,
  locked: boolean,
): Array<FormField> =>
  fields.map((field) =>
    locked && !["active", "payload", "base64Encoded", "latitude", "longitude", "altitude"].includes(field.name)
      ? { ...field, disabled: true, toolbar: undefined }
      : field,
  );

export const DeviceForm = (props: DeviceFormProps) => {
  // Hooks
  const formLogicRef = useRef<FormLogic | null>(null);
  const hydratedDeviceIdRef = useRef<string | null>(null);
  const mapLogic = useSensorMap({ mode: SensorMapMode.Coords });
  const { setAntennaRange, updatePos } = mapLogic;

  // Callbacks
  const setRegionDependentFields = useCallback(
    (
      logic: FormLogic,
      region: DeviceRegion,
      values?: {
        rx1DRO?: string;
        rx2DR?: number;
        rx2CF?: string;
      },
    ) => {
      logic.setProp("rx1DRO", "options", getRx1DataRateOptions(region));
      logic.setProp("rx2DR", "options", getRx2DataRateOptions(region));

      const defaultFrequency = rx2Frequency[region].toString();
      logic.setProp("rx2CF", "placeholder", defaultFrequency);
      logic.setValue("rx2CF", values?.rx2CF ?? defaultFrequency);

      if (values?.rx1DRO !== undefined) {
        logic.setValue("rx1DRO", values.rx1DRO);
      }

      if (values?.rx2DR !== undefined) {
        logic.setValue("rx2DR", values.rx2DR);
      }
    },
    [],
  );

  const hydrateForm = useCallback(
    (logic: FormLogic, device: Device) => {
      const values: Record<string, FormValue> = {
        active: device.active,
        name: device.name,
        devEUI: device.devEUI,
        latitude: device.locationConfig.latitude.toString(),
        longitude: device.locationConfig.longitude.toString(),
        altitude: device.locationConfig.altitude.toString(),
        region: device.locationConfig.region,
        class: device.class,
        activation: device.activation,
        radioProtocol: device.radioProtocol ?? RadioProtocol.Lorawan,
        radioReceiveMode: device.radioReceiveMode ?? RadioReceiveMode.Lorawan,
        joinEUI: device.OOTAConfig?.joinEUI ?? "",
        appKey: device.OOTAConfig?.appKey ?? "",
        devAddr: device.ABPConfig?.devAddr ?? "",
        nwkSKey: device.ABPConfig?.nwkSKey ?? "",
        appSKey: device.ABPConfig?.appSKey ?? "",
        rx1Delay: device.RX1Config.delay.toString(),
        rx1Duration: device.RX1Config.duration.toString(),
        rx1DRO: device.RX1Config.dataRateOffset.toString(),
        rx2Delay: device.RX2Config.delay.toString(),
        rx2Duration: device.RX2Config.duration.toString(),
        rx2CF: device.RX2Config.channelFrequency.toString(),
        rx2DR: device.RX2Config.dataRate,
        rx2ACKT: device.RX2Config.ACKTimeout.toString(),
        fPort: device.frameConfig.fPort.toString(),
        retransmission: device.frameConfig.retransmission.toString(),
        fCntUp: device.frameConfig.FCntUp?.toString() ?? "",
        fCntDown: device.frameConfig.FCntDown?.toString() ?? "",
        disableFrameCounterValidation:
          device.frameConfig.disableFrameCounterValidation,
        uplinkInterval: device.payloadConfig.uplinkInterval.toString(),
        oversizedPayloadBehavior: device.payloadConfig.oversizedPayloadBehavior,
        MType: device.payloadConfig.MType,
        payload: device.payloadConfig.payload,
        base64Encoded: device.payloadConfig.base64Encoded,
        payloadCodecEnabled:
          device.payloadConfig.payloadCodec?.enabled ?? false,
        payloadCodec:
          device.payloadConfig.payloadCodec?.script?.trim() ||
          defaultPayloadCodec,
        antennaRange: device.advancedConfig.antennaRange.toString(),
        ADREnabled: device.advancedConfig.ADREnabled,
      };
      Object.entries(values).forEach(([name, value]) =>
        logic.setValue(name, value),
      );

      setRegionDependentFields(logic, device.locationConfig.region, {
        rx1DRO: values.rx1DRO as string,
        rx2DR: values.rx2DR as number,
        rx2CF: values.rx2CF as string,
      });

      updatePos(
        device.locationConfig.latitude,
        device.locationConfig.longitude,
      );
      setAntennaRange(device.advancedConfig.antennaRange);
    },
    [setAntennaRange, updatePos, setRegionDependentFields],
  );

  // Effects
  useEffect(() => {
    if (
      props.device &&
      formLogicRef.current &&
      hydratedDeviceIdRef.current !== props.device.id
    ) {
      hydrateForm(formLogicRef.current, props.device);
      hydratedDeviceIdRef.current = props.device.id;
    }
  }, [hydrateForm, props.device]);

  useEffect(() => {
    if (formLogicRef.current && mapLogic.selectedPos) {
      if (
        formLogicRef.current.fieldsState.latitude.value !=
        mapLogic.selectedPos.lat.toString()
      )
        formLogicRef.current.setValue(
          "latitude",
          mapLogic.selectedPos.lat.toString(),
        );

      if (
        formLogicRef.current.fieldsState.longitude.value !=
        mapLogic.selectedPos.lng.toString()
      )
        formLogicRef.current.setValue(
          "longitude",
          mapLogic.selectedPos.lng.toString(),
        );

      const region = estimateRegion(
        mapLogic.selectedPos.lat,
        mapLogic.selectedPos.lng,
      );

      if (region) {
        formLogicRef.current.setValue("region", region);
      }
    }
  }, [mapLogic.selectedPos]);

  return (
    <div className="page-content device-form">
      <div className="form-wrapper">
        <Form
          onSubmit={props.onSubmit}
          submitButton={{
            value: props.device ? "Save" : "Create",
            className: "submit-button",
          }}
          onLogicReady={(logic) => {
            formLogicRef.current = logic;
            if (
              props.device &&
              hydratedDeviceIdRef.current !== props.device.id
            ) {
              hydrateForm(logic, props.device);
              hydratedDeviceIdRef.current = props.device.id;
            }
          }}
          fields={lockFieldsDuringSimulation(
            [
              booleanCheckboxField({
                value: props.device?.active ?? false,
                name: "active",
                label: "Active",
                error: null,
                text: "Enable",
                display: !!props.device,
              }),
              textField({
                name: "name",
                label: "Name",
                value: "",
                placeholder: "Device Name",
                error: null,
                required: true,
              }),
              textField({
                name: "devEUI",
                label: "DevEUI",
                value: "",
                placeholder: "70B3D57ED0001234",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 16)
                    .toUpperCase();
                },
                info: {
                  default:
                    "Unique identifier assigned to the LoRaWAN end device",
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{16}$/,
                    error: "Invalid DevEUI",
                  },
                ],
                error: null,
                required: true,
                disabled: !!props.device,
                toolbar: props.device ? undefined : (
                  <Button
                    value={
                      <span className="material-symbols-outlined">cached</span>
                    }
                    layout={ButtonLayout.Icon}
                    type={ButtonType.Outlined}
                    onClick={() => {
                      formLogicRef.current?.setValue("devEUI", randomDevEUI());
                    }}
                  />
                ),
              }),
              textField({
                name: "latitude",
                label: "Latitude",
                placeholder: "45.4642",
                value: "",
                error: null,
                info: { default: "Decimal degrees (-90 to 90)" },
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9.-]/g, "")
                    .replace(/(?!^)-/g, "")
                    .replace(/(\..*)\./g, "$1");
                },
                validations: [
                  {
                    rule: latValidation,
                    error: "Invalid latitude",
                  },
                ],
                required: true,
                onChange: (logic: FormLogic) => {
                  if (
                    logic.fieldsState.latitude.value &&
                    (logic.fieldsState.latitude.value as string).match(
                      latValidation,
                    ) &&
                    logic.fieldsState.longitude.value &&
                    (logic.fieldsState.longitude.value as string).match(
                      lngValidation,
                    )
                  ) {
                    mapLogic.updatePos(
                      Number(logic.fieldsState.latitude.value),
                      Number(logic.fieldsState.longitude.value),
                    );
                  }
                },
              }),
              textField({
                name: "longitude",
                label: "Longitude",
                placeholder: "9.1900",
                value: "",
                error: null,
                info: { default: "Decimal degrees (-180 to 180)" },
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9.-]/g, "")
                    .replace(/(?!^)-/g, "")
                    .replace(/(\..*)\./g, "$1");
                },
                validations: [
                  {
                    rule: lngValidation,
                    error: "Invalid longitude",
                  },
                ],
                required: true,
                onChange: (logic: FormLogic) => {
                  if (
                    logic.fieldsState.latitude.value &&
                    (logic.fieldsState.latitude.value as string).match(
                      latValidation,
                    ) &&
                    logic.fieldsState.longitude.value &&
                    (logic.fieldsState.longitude.value as string).match(
                      lngValidation,
                    )
                  ) {
                    mapLogic.updatePos(
                      Number(logic.fieldsState.latitude.value),
                      Number(logic.fieldsState.longitude.value),
                    );
                  }
                },
              }),
              textField({
                name: "altitude",
                label: "Altitude",
                placeholder: "120",
                value: "",
                error: null,
                info: { default: "Meters above sea level" },
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9.-]/g, "")
                    .replace(/(?!^)-/g, "")
                    .replace(/(\..*)\./g, "$1");
                },
                validations: [
                  {
                    rule: /^-?\d+(?:\.\d+)?$/,
                    error: "Invalid altitude",
                  },
                ],
                required: true,
                toolbar: (
                  <Button
                    value={
                      <span className="material-symbols-outlined">cached</span>
                    }
                    layout={ButtonLayout.Icon}
                    type={ButtonType.Outlined}
                    onClick={async () => {
                      if (
                        !formLogicRef.current ||
                        !(
                          formLogicRef.current.fieldsState.longitude
                            .value as string
                        ).match(lngValidation) ||
                        !(
                          formLogicRef.current.fieldsState.latitude
                            .value as string
                        ).match(latValidation)
                      ) {
                        NotificationHandler.instance.error(
                          "Select valid latitude and longitude",
                        );
                        return;
                      }
                      const res = await getAltitude(
                        Number(formLogicRef.current.fieldsState.latitude.value),
                        Number(
                          formLogicRef.current.fieldsState.longitude.value,
                        ),
                      );

                      formLogicRef.current.setValue("altitude", res.toString());
                    }}
                  />
                ),
              }),
              selectField({
                value: null,
                name: "region",
                label: "Region",
                error: null,
                placeholder: "Select a region",
                options: Object.values(DeviceRegion).map((reg: string) => {
                  return {
                    value: reg,
                  };
                }),
                info: { default: "LoRaWAN regional frequency plan" },
                onChange: (logic: FormLogic) => {
                  logic.setValue("rx1DRO", null);

                  if (logic.fieldsState.region.value) {
                    setRegionDependentFields(
                      logic,
                      logic.fieldsState.region.value as DeviceRegion,
                    );
                  } else {
                    logic.setProp(
                      "rx1DRO",
                      "options",
                      [] as Array<SelectOption>,
                    );
                    logic.setProp(
                      "rx2DR",
                      "options",
                      [] as Array<SelectOption>,
                    );
                  }
                },
                required: true,
                toolbar: (
                  <Button
                    value={
                      <span className="material-symbols-outlined">cached</span>
                    }
                    layout={ButtonLayout.Icon}
                    type={ButtonType.Outlined}
                    onClick={async () => {
                      if (
                        !formLogicRef.current ||
                        !(
                          formLogicRef.current.fieldsState.longitude
                            .value as string
                        ).match(lngValidation) ||
                        !(
                          formLogicRef.current.fieldsState.latitude
                            .value as string
                        ).match(latValidation)
                      ) {
                        NotificationHandler.instance.error(
                          "Select valid latitude and longitude",
                        );
                        return;
                      }

                      const res = estimateRegion(
                        Number(formLogicRef.current.fieldsState.latitude.value),
                        Number(
                          formLogicRef.current.fieldsState.longitude.value,
                        ),
                      );

                      if (!res)
                        NotificationHandler.instance.error(
                          "Unable to estimate region",
                        );

                      formLogicRef.current.setValue("region", res);
                    }}
                  />
                ),
              }),
              radioField({
                value: null,
                name: "class",
                label: "Class",
                error: null,
                options: Object.keys(DeviceClass).map((key: string) => {
                  return {
                    value: DeviceClass[key as keyof typeof DeviceClass],
                    displayed: <>{key.replace("Class", "")}</>,
                  };
                }),
                info: {
                  default: "Device class",
                  [DeviceClass.ClassA]:
                    "Lowest power consumption, with downlink windows after each uplink",
                  [DeviceClass.ClassB]:
                    "Scheduled downlink windows, offering predictable network communication",
                  [DeviceClass.ClassC]:
                    "Nearly continuous downlink availability, with highest power consumption",
                },
                required: true,
                disabled: !!props.device,
              }),
              radioField({
                value: null,
                name: "activation",
                label: "Activation Type",
                error: null,
                options: Object.keys(DeviceActivation).map((key: string) => {
                  return {
                    value:
                      DeviceActivation[key as keyof typeof DeviceActivation],
                    displayed: <>{key.toUpperCase()}</>,
                  };
                }),
                info: {
                  default: "Device Activation Type",
                  [DeviceActivation.Oota]:
                    "Secure activation through network join using unique session credentials",
                  [DeviceActivation.Abp]:
                    "Direct activation using preconfigured device and session credentials",
                },
                required: true,
                disabled: !!props.device,
              }),
              selectField({
                value: RadioProtocol.Lorawan,
                name: "radioProtocol",
                label: "Radio Protocol",
                error: null,
                options: [
                  { value: RadioProtocol.Lorawan, displayed: <>LoRaWAN</> },
                  { value: RadioProtocol.RawLora, displayed: <>Raw LoRa</> },
                ],
                info: {
                  default:
                    "Protocol used by the device when transmitting radio frames",
                  [RadioProtocol.Lorawan]:
                    "Adds the LoRaWAN network and security headers",
                  [RadioProtocol.RawLora]:
                    "Transmits the payload as a raw LoRa frame without LoRaWAN headers",
                },
                required: true,
              }),
              selectField({
                value: RadioReceiveMode.Lorawan,
                name: "radioReceiveMode",
                label: "Radio Receive Mode",
                error: null,
                options: [
                  {
                    value: RadioReceiveMode.Lorawan,
                    displayed: <>LoRaWAN</>,
                  },
                  {
                    value: RadioReceiveMode.RawLora,
                    displayed: <>Raw LoRa</>,
                  },
                  { value: RadioReceiveMode.Auto, displayed: <>Auto</> },
                ],
                info: {
                  default:
                    "Frame protocols accepted by the device while receiving",
                  [RadioReceiveMode.Lorawan]:
                    "Accepts only structurally valid LoRaWAN frames",
                  [RadioReceiveMode.RawLora]:
                    "Accepts only raw LoRa frames",
                  [RadioReceiveMode.Auto]:
                    "Accepts LoRaWAN frames and falls back to raw LoRa",
                },
                required: true,
              }),
              textField({
                name: "joinEUI",
                label: "JoinEUI",
                value: "",
                placeholder: "0004A30B001C0530",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 16)
                    .toUpperCase();
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{16}$/,
                    error: "Invalid JoinEUI",
                  },
                ],
                info: {
                  default:
                    "Identifies the application entity handling the device join",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.activation.value == DeviceActivation.Oota,
              }),
              textField({
                name: "appKey",
                label: "AppKey",
                value: "",
                placeholder: "2B7E151628AED2A6ABF7158809CF4F3C",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 32)
                    .toUpperCase();
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{32}$/,
                    error: "Invalid AppKey",
                  },
                ],
                info: {
                  default:
                    "Secret key used to securely authenticate an OTAA join",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.activation.value == DeviceActivation.Oota,
              }),
              textField({
                name: "devAddr",
                label: "DevAddr",
                value: "",
                placeholder: "26011BDA",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 8)
                    .toUpperCase();
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{8}$/,
                    error: "Invalid DevAddr",
                  },
                ],
                info: {
                  default:
                    "32-bit address assigned to the device during ABP activation",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.activation.value == DeviceActivation.Abp,
              }),
              textField({
                name: "nwkSKey",
                label: "NwkSKey",
                value: "",
                placeholder: "00112233445566778899AABBCCDDEEFF",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 32)
                    .toUpperCase();
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{32}$/,
                    error: "Invalid NwkSKey",
                  },
                ],
                info: {
                  default:
                    "Session key used to secure LoRaWAN network-layer communication",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.activation.value == DeviceActivation.Abp,
              }),
              textField({
                name: "appSKey",
                label: "AppSKey",
                value: "",
                placeholder: "0004AFFEEDDCCBBAA9988776655443322110030B001C0530",
                format: (raw: string) => {
                  return raw
                    .replace(/[^0-9A-Fa-f]/g, "")
                    .slice(0, 32)
                    .toUpperCase();
                },
                validations: [
                  {
                    rule: /^[0-9A-F]{32}$/,
                    error: "Invalid AppSKey",
                  },
                ],
                info: {
                  default:
                    "Session key used to encrypt application payload data",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.activation.value == DeviceActivation.Abp,
              }),
              textField({
                name: "rx1Delay",
                label: "RX1 Delay",
                value: "",
                placeholder: "1000",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Delay before opening the RX1 downlink reception window in milliseconds",
                },
                error: null,
                required: true,
              }),
              textField({
                name: "rx1Duration",
                label: "RX1 Duration",
                value: "",
                placeholder: "3000",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Duration of the RX1 downlink reception window in milliseconds",
                },
                error: null,
                required: true,
              }),
              selectField({
                value: null,
                name: "rx1DRO",
                label: "RX1 Data Rate offset",
                error: null,
                placeholder: "Select DRO",
                options: props.device
                  ? getRx1DataRateOptions(props.device.locationConfig.region)
                  : [],
                info: {
                  default: "Offset applied to the uplink data rate for RX1",
                },
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  !!fieldsState.region.value,
              }),
              textField({
                name: "rx2Delay",
                label: "RX2 Delay",
                value: "",
                placeholder: "1000",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Delay before opening the RX2 downlink reception window in milliseconds",
                },
                error: null,
                required: true,
              }),
              textField({
                name: "rx2Duration",
                label: "RX2 Duration",
                value: "",
                placeholder: "3000",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Duration of the RX2 downlink reception window in milliseconds",
                },
                error: null,
                required: true,
              }),
              textField({
                name: "rx2CF",
                label: "RX2 Channel Frequency",
                value: "",
                placeholder: "",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Frequency used for receiving downlink messages during RX2, in Hz",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  !!fieldsState.region.value,
              }),
              selectField({
                name: "rx2DR",
                label: "RX2 Data Rate",
                value: "",
                placeholder: "0",
                options: props.device
                  ? getRx2DataRateOptions(props.device.locationConfig.region)
                  : [],
                info: {
                  default:
                    "Data rate used for receiving downlink messages during RX2",
                },
                error: null,
                required: true,
                display: (fieldsState: Record<string, FormField>) =>
                  !!fieldsState.region.value,
              }),
              textField({
                name: "rx2ACKT",
                label: "RX2 ACK Timeout",
                value: "",
                placeholder: "2000",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                validations: [
                  {
                    rule: /^(?:1000|[12][0-9]{3}|3000)$/,
                    error: "Value must be between 1000 and 3000",
                  },
                ],
                info: {
                  default:
                    "Time the device waits for an acknowledgement after a confirmed uplink",
                },
                error: null,
                required: true,
              }),
              textField({
                name: "fPort",
                label: "FPort",
                value: "",
                placeholder: "1",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default: "Application port used to identify the payload type",
                },
                validations: [
                  {
                    rule: /^(?:[1-9]|[1-9][0-9]|1[0-9]{2}|2[0-2][0-9]|23[0-3])$/,
                    error: "Fport must be between 1 and 223",
                  },
                ],
                error: null,
                required: true,
              }),
              textField({
                name: "retransmission",
                label: "Retransmission",
                value: "",
                placeholder: "0",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Number of times an uplink is retransmitted after transmission failure",
                },
                error: null,
                required: true,
              }),
              textField({
                name: "fCntUp",
                label: "FCntUp",
                value: "",
                placeholder: "0",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Uplink frame counter used to track transmitted frames",
                },
                error: null,
              }),
              textField({
                name: "fCntDown",
                label: "FCntDown",
                value: "",
                placeholder: "0",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                info: {
                  default:
                    "Downlink frame counter used to track received frames",
                },
                error: null,
              }),
              booleanCheckboxField({
                value: false,
                name: "disableFrameCounterValidation",
                label: "Frame Counter Validation",
                error: null,
                text: "Disable",
                info: {
                  default: "Disables validation of downlink frame counters",
                },
              }),
              textField({
                name: "uplinkInterval",
                label: "Uplink Interval",
                value: "",
                placeholder: "10",
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                validations: [
                  {
                    rule: /^[1-9][0-9]*$/,
                    error: "Uplink Interval must be greater than 0",
                  },
                ],
                info: {
                  default:
                    "Time between consecutive uplink transmissions, in seconds",
                },
                error: null,
                required: true,
              }),
              radioField({
                value: null,
                name: "oversizedPayloadBehavior",
                label: "Oversized Payload Behavior",
                error: null,
                options: Object.keys(OversizedPayloadBehavior).map(
                  (key: string) => {
                    return {
                      value:
                        OversizedPayloadBehavior[
                          key as keyof typeof OversizedPayloadBehavior
                        ],
                      displayed: <>{key}</>,
                    };
                  },
                ),
                info: {
                  default:
                    "Defines how payloads exceeding the maximum size are handled",
                  [OversizedPayloadBehavior.Fragment]:
                    "Splits oversized payloads into multiple smaller transmissions",
                  [OversizedPayloadBehavior.Truncate]:
                    "Truncates oversized payloads to the maximum allowed size",
                },
                required: true,
              }),
              radioField({
                value: null,
                name: "MType",
                label: "MType",
                error: null,
                options: Object.keys(DeviceMType).map((key: string) => {
                  return {
                    value: DeviceMType[key as keyof typeof DeviceMType],
                    displayed: (
                      <>
                        {DeviceMType[
                          key as keyof typeof DeviceMType
                        ].replaceAll("-", " ")}
                      </>
                    ),
                  };
                }),
                info: {
                  default:
                    "Defines the LoRaWAN message type used for uplink transmissions",
                  [DeviceMType.ConfirmedDataUp]:
                    "Requires a downlink acknowledgement from the network",
                  [DeviceMType.UnconfirmedDataUp]:
                    "Does not require a downlink acknowledgement from the network",
                },
                required: true,
              }),
              textField({
                name: "payload",
                label: "Payload",
                value: "",
                placeholder: "Hello LoRaWAN",
                info: {
                  default: "Application payload transmitted by the device",
                },
                error: null,
                required: true,
              }),
              booleanCheckboxField({
                value: false,
                name: "base64Encoded",
                label: "Base64 Encoded",
                error: null,
                text: "Enable",
                info: {
                  default: "Application payload transmitted by the device",
                },
                onChange: (logic: FormLogic) => {
                  if (logic.fieldsState.base64Encoded.value) {
                    logic.setProp(
                      "payload",
                      "placeholder",
                      "SGVsbG8gTG9SQVdBTg==",
                    );
                    logic.setProp("payload", "validations", [
                      {
                        rule: /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/,
                        error: "Enter a valid Base64 value",
                      },
                    ]);
                  } else {
                    logic.setProp("payload", "placeholder", "Hello LoRaWAN");
                    logic.setProp("payload", "validations", undefined);
                  }
                },
              }),
              booleanCheckboxField({
                value: false,
                name: "payloadCodecEnabled",
                label: "Payload Codec",
                error: null,
                text: "Enable",
                info: {
                  default:
                    "Uses a JavaScript decoder to convert uplink bytes into readable data",
                },
                onChange: (logic: FormLogic) => {
                  if (!logic.fieldsState.payloadCodec.value) {
                    logic.setValue("payloadCodec", defaultPayloadCodec);
                  }
                },
              }),
              codeEditorField({
                name: "payloadCodec",
                label: "JavaScript Payload Codec",
                value: defaultPayloadCodec,
                placeholder: defaultPayloadCodec,
                info: {
                  default:
                    "Provide a decodeUplink(input) function compatible with the payload codec API",
                },
                error: null,
                required: true,
                validations: [
                  {
                    rule: (value: FormValue) =>
                      isValidJavaScript(String(value)),
                    error: "Enter valid JavaScript syntax",
                  },
                  {
                    rule: (value: FormValue) =>
                      !isValidJavaScript(String(value)) || hasPayloadDecoder(String(value)),
                    error: "Define a decodeUplink(input) function",
                    realtime: true,
                  },
                ],
                resize: true,
                display: (fieldsState: Record<string, FormField>) =>
                  fieldsState.payloadCodecEnabled.value === true,
              }),
              textField({
                name: "antennaRange",
                label: "Antenna Range",
                value: "",
                placeholder: "10000",
                info: {
                  default:
                    "Maximum distance within which the device can communicate, in meters",
                },
                format: (raw: string) => {
                  return raw.replace(/[^0-9]/g, "");
                },
                validations: [
                  {
                    rule: /^[1-9][0-9]*$/,
                    error: "Antenna Range must be greater than 0",
                  },
                ],
                onChange: (logic: FormLogic) => {
                  mapLogic.setAntennaRange(
                    logic.fieldsState["antennaRange"].value
                      ? Number(logic.fieldsState["antennaRange"].value)
                      : null,
                  );
                },
                error: null,
                required: true,
              }),
              booleanCheckboxField({
                value: false,
                name: "ADREnabled",
                label: "Adaptive Data Rate",
                error: null,
                text: "Enable",
                info: {
                  default:
                    "Allows the network to optimize the device's data rate and transmission power",
                },
              }),
            ],
            !!props.device && props.simulationActive,
          )}
        />
      </div>
      <SensorMap logic={mapLogic} mode={SensorMapMode.Coords} />
    </div>
  );
};
