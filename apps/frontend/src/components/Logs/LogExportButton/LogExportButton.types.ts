/** Options for downloading simulation logs belonging to the current profile. */
export interface LogExportButtonProps {
  /** Run identifier. Omit to export data for the active profile. */
  runID?: string;
  /** Downloads aggregated telemetry in place of the raw log export. */
  telemetry?: boolean;
  /** Prevents exporting before the history has loaded or when it is empty. */
  disabled?: boolean;
}
