/**
 * Properties accepted by the launcher connection form.
 */
export interface ConnectionFormProps {
  /** Notifies the page when a connection attempt starts or ends. */
  onLoadingChange: (loadingText: string | null) => void;
}
