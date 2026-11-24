"use client";

import { useEffect, useRef, useState } from "react";
import { PopupLogic, UsePopupProps } from "./Popup.types";

export const usePopup = (props: UsePopupProps): PopupLogic => {
  // States
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [isVisible, setIsVisible] = useState<boolean>(false);

  // Hooks
  const closeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  // Callbacks
  const openPopup = () => {
    clearTimeout(closeTimer.current);
    setIsOpen(true);
    setIsVisible(true);
  };

  const closePopup = () => {
    clearTimeout(closeTimer.current);
    setIsVisible(false);
    closeTimer.current = setTimeout(() => {
      setIsOpen(false);
    }, 300);
  };

  // Effects
  useEffect(() => () => clearTimeout(closeTimer.current), []);

  return {
    isOpen,
    isVisible,
    openPopup,
    closePopup,
  };
};
