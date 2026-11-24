"use client";

import { useCallback } from "react";
import { useSimulationService } from "@lwn-simulator/sdk";
import { Button, ButtonType, NotificationHandler } from "@lwn-simulator/ui-components";
import { LogExportButtonProps } from "./LogExportButton.types";

export const LogExportButton = (props: LogExportButtonProps) => {
  // Hooks
  const { getLogExportURL, getTelemetryExportURL } = useSimulationService();
  const { getExportURL, dataName, filename, label } = props.telemetry
    ? {
        getExportURL: getTelemetryExportURL,
        dataName: "simulation telemetry",
        filename: props.runID ? `simulation-${props.runID}-telemetry.zip` : "simulation-telemetry-summary.csv",
        label: props.runID ? "Export telemetry" : "Export telemetry summary",
      }
    : {
        getExportURL: getLogExportURL,
        dataName: "simulation logs",
        filename: props.runID ? `simulation-${props.runID}.json` : "simulation-logs.zip",
        label: props.runID ? "Export JSON" : "Export all (ZIP)",
      };

  // Callbacks
  const download = useCallback(async () => {
    try {
      const response = await fetch(getExportURL(props.runID));
      if (!response.ok) {
        throw new Error(`Unable to export ${dataName}. Please try again.`);
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename;
      anchor.hidden = true;
      document.body.appendChild(anchor);
      anchor.click();
      window.setTimeout(() => {
        anchor.remove();
        URL.revokeObjectURL(url);
      }, 1000);
    } catch (error) {
      NotificationHandler.instance.error(
        error instanceof Error ? error.message : `Unable to export ${dataName}.`,
      );
    }
  }, [dataName, filename, getExportURL, props.runID]);

  return (
    <Button
      className="log-export-button"
      value={label}
      type={ButtonType.Outlined}
      disabled={props.disabled}
      onClick={download}
    />
  );
};
