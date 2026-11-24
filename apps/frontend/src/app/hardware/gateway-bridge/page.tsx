"use client";

import {
  Button,
  ButtonType,
  Card,
  CardLayout,
  Form,
  FormLogic,
  FormValue,
  NotificationHandler,
  PageHeader,
  booleanCheckboxField,
  textField,
} from "@lwn-simulator/ui-components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  useProfileConfigurationService,
} from "@lwn-simulator/sdk";
import {
  defaultGatewayBridgeConfig,
} from "@/utils/gatewayBridge";
import {
  GatewayBridgeFormState,
  GatewayBridgeFormValueMap,
  GatewayBridgeFormValues,
} from "./page.types";
import "./gateway-bridge.scss";

const defaultFormValues: GatewayBridgeFormValues = {
  enabled: defaultGatewayBridgeConfig.enabled,
  address: defaultGatewayBridgeConfig.address ?? "0.0.0.0",
  port: defaultGatewayBridgeConfig.port ?? 1700,
};

const toConfig = (values: GatewayBridgeFormValueMap) => ({
  enabled: values.enabled === true,
  address: String(values.address ?? "0.0.0.0").trim(),
  port: Number(values.port ?? 1700),
});

const GatewayBridgePage = () => {
  const profileConfiguration = useProfileConfigurationService();
  const initialValues = useMemo(
    () => {
      if (profileConfiguration.gatewayBridgeConfig) {
        const config = profileConfiguration.gatewayBridgeConfig;
        return {
          enabled: config.enabled,
          address: config.address ?? defaultGatewayBridgeConfig.address ?? "0.0.0.0",
          port: config.port ?? defaultGatewayBridgeConfig.port ?? 1700,
        };
      }
      return defaultFormValues;
    },
    [
      profileConfiguration.gatewayBridgeConfig,
    ],
  );
  const logicRef = useRef<FormLogic | null>(null);
  const [state, setState] = useState<GatewayBridgeFormState>({
    logic: null,
    values: defaultFormValues,
    lastSaved: defaultFormValues,
  });

  useEffect(() => {
    const values = initialValues;
    setState((current) => ({ ...current, values, lastSaved: values }));
    logicRef.current?.setValue("enabled", values.enabled);
    logicRef.current?.setValue("address", values.address);
    logicRef.current?.setValue("port", String(values.port));
  }, [initialValues]);

  const fields = useMemo(
    () => [
      booleanCheckboxField({
        name: "enabled",
        label: "Gateway Bridge",
        value: initialValues.enabled,
        error: null,
        text: "Enable the Semtech UDP bridge",
      }),
      textField({
        name: "address",
        label: "Gateway Bridge's address",
        value: initialValues.address,
        error: null,
        placeholder: "IPv4 or URL",
        required: true,
        disabled: (fieldsState) => fieldsState.enabled.value !== true,
      }),
      textField({
        name: "port",
        label: "Gateway Bridge's Port",
        value: String(initialValues.port),
        error: null,
        placeholder: "1700",
        required: true,
        disabled: (fieldsState) => fieldsState.enabled.value !== true,
        format: (raw) => raw.replace(/[^0-9]/g, ""),
      }),
    ],
    [initialValues],
  );

  const handleLogicReady = useCallback((logic: FormLogic) => {
    logicRef.current = logic;
  }, []);

  const handleSubmit = useCallback(
    async (values: Record<string, FormValue>) => {
      try {
        const config = await profileConfiguration.updateGatewayBridge(
          toConfig(values),
        );
        const nextValues = {
          enabled: config.enabled,
          address: config.address ?? "0.0.0.0",
          port: config.port ?? 1700,
        };
        setState((current) => ({
          ...current,
          values: nextValues,
          lastSaved: nextValues,
        }));
        NotificationHandler.instance.success("Gateway Bridge settings saved");
      } catch (error) {
        NotificationHandler.instance.error(
          error instanceof Error ? error.message : "Unable to save settings",
        );
      }
    },
    [profileConfiguration],
  );

  const reset = useCallback(async () => {
    try {
      const values = await profileConfiguration.updateGatewayBridge(
        defaultGatewayBridgeConfig,
      );
      logicRef.current?.setValue("enabled", values.enabled);
      logicRef.current?.setValue("address", values.address ?? "0.0.0.0");
      logicRef.current?.setValue("port", String(values.port ?? 1700));
      setState((current) => ({
        ...current,
        values: {
          enabled: values.enabled,
          address: values.address ?? "0.0.0.0",
          port: values.port ?? 1700,
        },
        lastSaved: {
          enabled: values.enabled,
          address: values.address ?? "0.0.0.0",
          port: values.port ?? 1700,
        },
      }));
      NotificationHandler.instance.success("Gateway Bridge settings reset");
    } catch (error) {
      NotificationHandler.instance.error(
        error instanceof Error ? error.message : "Unable to reset settings",
      );
    }
  }, [profileConfiguration]);

  const isEnabled = state.values.enabled;
  const endpoint = `${state.values.address}:${state.values.port}`;

  return (
    <main className="page gateway-bridge-page">
      <PageHeader
        title="Gateway Bridge"
        subTitle="Configure the UDP boundary used by real gateways and ChirpStack Gateway Bridge"
      >
        <Button
          value="Back to gateways"
          type={ButtonType.Outlined}
          onClick={() => window.history.back()}
        />
      </PageHeader>

      <section className="gateway-bridge-page__grid">
        <Card className="gateway-bridge-page__configuration" layout={CardLayout.Padded}>
          <div className="gateway-bridge-page__section-heading">
            <div>
              <p className="gateway-bridge-page__eyebrow">Configuration</p>
              <h2>UDP bridge endpoint</h2>
              <p>These settings are applied the next time a simulation starts.</p>
            </div>
            <span className={`gateway-bridge-page__status${isEnabled ? " is-enabled" : ""}`}>
              <span /> {isEnabled ? "Enabled" : "Disabled"}
            </span>
          </div>
          <Form
            fields={fields}
            onLogicReady={handleLogicReady}
            onSubmit={handleSubmit}
            submitButton={{ value: "Save settings" }}
          />
          <Button
            value="Reset to default"
            type={ButtonType.Outlined}
            onClick={reset}
          />
        </Card>

        <Card className="gateway-bridge-page__overview" layout={CardLayout.Padded}>
          <div className="gateway-bridge-page__section-heading">
            <div>
              <p className="gateway-bridge-page__eyebrow">Runtime endpoint</p>
              <h2>Connection details</h2>
            </div>
            <span className="material-symbols-outlined gateway-bridge-page__icon">router</span>
          </div>
          <dl className="gateway-bridge-page__details">
            <div>
              <dt>Protocol</dt>
              <dd>Semtech UDP</dd>
            </div>
            <div>
              <dt>Bridge address</dt>
              <dd className="gateway-bridge-page__mono">{endpoint}</dd>
            </div>
            <div>
              <dt>Default port</dt>
              <dd>{state.values.port} / UDP</dd>
            </div>
            <div>
              <dt>Last saved</dt>
              <dd>{state.lastSaved ? "Saved to profile" : "Not saved yet"}</dd>
            </div>
          </dl>
        </Card>
      </section>

      <section className="gateway-bridge-page__grid gateway-bridge-page__grid--lower">
        <Card className="gateway-bridge-page__guide" layout={CardLayout.Padded}>
          <p className="gateway-bridge-page__eyebrow">How it works</p>
          <h2>Connect a real gateway</h2>
          <ol>
            <li>Enable the bridge and save the UDP listen address.</li>
            <li>Configure the packet forwarder or ChirpStack Gateway Bridge to target this endpoint.</li>
            <li>Start a simulation with at least one active real gateway.</li>
          </ol>
          <p className="gateway-bridge-page__note">
            Virtual gateways remain inside the simulator and do not use this UDP endpoint.
          </p>
        </Card>
        <Card className="gateway-bridge-page__guide" layout={CardLayout.Padded}>
          <p className="gateway-bridge-page__eyebrow">Supported traffic</p>
          <h2>Semtech packet-forwarder flow</h2>
          <div className="gateway-bridge-page__capabilities">
            <span><span className="material-symbols-outlined">north</span> Uplinks and acknowledgements</span>
            <span><span className="material-symbols-outlined">south</span> Timed downlinks</span>
            <span><span className="material-symbols-outlined">favorite</span> Heartbeats and reconnects</span>
          </div>
        </Card>
      </section>
    </main>
  );
};

export default GatewayBridgePage;
