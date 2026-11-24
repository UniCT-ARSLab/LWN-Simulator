"use client";

import { Card } from "@/components/layout";
import { PopupProps } from "./Popup.types";
import cn from "classnames";
import { KeyboardEvent, useEffect, useId, useRef } from "react";
import { useOutsideAlerter } from "@/hooks";

const Popup = (props: PopupProps) => {
  // Hooks
  const cardRef = useRef<HTMLDivElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const titleId = useId();

  // Callbacks
  const dismiss = () => {
    if (props.dismissible !== false && props.logic.isVisible) {
      props.logic.closePopup();
    }
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape") {
      event.stopPropagation();
      dismiss();
    }
    if (event.key !== "Tab") return;

    const focusable = Array.from(
      cardRef.current?.querySelectorAll<HTMLElement>(
        'button:enabled, input:enabled:not([type="hidden"]), select:enabled, textarea:enabled, a[href], [tabindex]:not([tabindex="-1"]):not(:disabled)',
      ) ?? [],
    ).filter((element) => element.getClientRects().length > 0);
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const current = document.activeElement;
    if (!first || !focusable.some((element) => element === current) ||
      (event.shiftKey ? current === first : current === last)) {
      event.preventDefault();
      (event.shiftKey ? last : first)?.focus();
    }
  };

  // Effects
  useOutsideAlerter({
    ref: cardRef,
    onClickOutside: dismiss,
  });

  useEffect(() => {
    if (!props.logic.isVisible) return;
    const previous = document.activeElement;
    dialogRef.current?.focus();
    return () => {
      if (previous instanceof HTMLElement) previous.focus();
    };
  }, [props.logic.isVisible]);

  return (
    <>
      {props.logic.isOpen && (
        <div
          ref={dialogRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          tabIndex={-1}
          onKeyDown={handleKeyDown}
          className={cn(
            "popup",
            props.logic.isVisible ? "visible" : "not-visible",
          )}
        >
          <Card className={cn("popup-card", props.className)} ref={cardRef}>
            <div className="popup-header">
              <strong id={titleId}>{props.title}</strong>
              <button
                type="button"
                className="material-symbols-outlined"
                aria-label="Close"
                disabled={props.dismissible === false}
                onClick={dismiss}
              >
                close
              </button>
            </div>
            <div className="popup-body">{props.children}</div>
          </Card>
        </div>
      )}
    </>
  );
};

export default Popup;
