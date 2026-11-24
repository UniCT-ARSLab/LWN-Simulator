/** Properties for the confirmation action on a hardware detail page. */
export interface DeleteHardwareButtonProps {
  /** Type of hardware to name in the confirmation. */
  kind: "device" | "gateway";
  /** Display name of the hardware being deleted. */
  name: string;
  /** Deletes the hardware and rejects if the request fails. */
  onDelete: () => Promise<void>;
  /** Runs after the deletion succeeds. */
  onDeleted: () => void;
}
