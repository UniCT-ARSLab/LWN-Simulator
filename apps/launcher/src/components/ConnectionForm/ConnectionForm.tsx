"use client";

import {
  Card,
  CardLayout,
  CardType,
  Form,
  FormField,
  FormValue,
  NotificationHandler,
  radioField,
  textField,
} from "@lwn-simulator/ui-components";
import { ConnectionFormProps } from "./ConnectionForm.types";

export const ConnectionForm = (props: ConnectionFormProps) => {
  // Callbacks
  const handleSubmit = async (
    values: Record<string, FormValue>,
  ): Promise<void> => {
    switch (values.env) {
      case "local":
        props.onLoadingChange("Running local backend");
        try {
          const result = await window.electron.connectLocal();

          if (!result.success) {
            props.onLoadingChange(null);
            NotificationHandler.instance.error(
              result.message || "Failed to start local backend",
            );
          }
        } catch {
          props.onLoadingChange(null);
          NotificationHandler.instance.error("Failed to start local backend");
        }
        break;
      case "remote":
        props.onLoadingChange(`Connecting to: ${values.url}`);
        try {
          const result = await window.electron.connectRemote(
            values.url as string,
          );
          if (!result.success) {
            props.onLoadingChange(null);
            NotificationHandler.instance.error(result.message || "Error");
          }
        } catch {
          props.onLoadingChange(null);
          NotificationHandler.instance.error("Failed to connect to remote server");
        }
        break;
      default:
        break;
    }
  };

  return (
    <Card
      type={CardType.Transparent}
      layout={CardLayout.Padded}
      className="connection-card"
    >
      <Form
        fields={[
          radioField({
            name: "env",
            label: "Server Environment",
            value: null,
            error: null,
            options: [
              {
                value: "local",
                displayed: <>Local Backend</>,
              },
              {
                value: "remote",
                displayed: <>Remote Server</>,
              },
            ],
            required: true,
          }),
          textField({
            name: "url",
            label: "Remote Url",
            value: "",
            placeholder: "https://example.com:8080",
            error: null,
            required: true,
            display: (fieldsState: Record<string, FormField>) =>
              fieldsState.env.value === "remote",
            validations: [
              {
                rule: /^(?:(?:https?):\/\/)?(?:localhost|(?:\d{1,3}\.){3}\d{1,3}|(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,})(?::\d{1,5})?$/,
                error: "Invalid url",
              },
            ],
          }),
        ]}
        onSubmit={handleSubmit}
      />
    </Card>
  );
};
