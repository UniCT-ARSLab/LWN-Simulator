"use client";

import { Button, ButtonType, Popup, usePopup } from "@lwn-simulator/ui-components";
import { useRef, useState } from "react";
import { DeleteHardwareButtonProps } from "./DeleteHardwareButton.types";

export const DeleteHardwareButton = (props: DeleteHardwareButtonProps) => {
  // States
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);

  // Hooks
  const popup = usePopup({});
  const inFlight = useRef(false);

  // Callbacks
  const confirmDelete = async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setPending(true);
    setFailed(false);

    try {
      await props.onDelete();
    } catch {
      setFailed(true);
      return;
    } finally {
      inFlight.current = false;
      setPending(false);
    }

    popup.closePopup();
    props.onDeleted();
  };

  return (
    <>
      <Button
        value="Delete"
        type={ButtonType.Outlined}
        onClick={() => {
          setFailed(false);
          popup.openPopup();
        }}
      />
      <Popup logic={popup} title={`Delete ${props.kind}`} dismissible={!pending}>
        <div className="delete-hardware-dialog">
          <p>
            Delete {props.kind} <strong>{props.name}</strong>? This action cannot
            be undone.
          </p>
          {failed && <p role="alert">Failed to delete this {props.kind}. Try again.</p>}
          <div className="delete-hardware-dialog-actions">
            <Button
              value="Cancel"
              type={ButtonType.Outlined}
              onClick={popup.closePopup}
              disabled={pending}
            />
            <Button
              value={`Delete ${props.kind}`}
              onClick={confirmDelete}
              disabled={pending}
              loading={pending}
            />
          </div>
        </div>
      </Popup>
    </>
  );
};
