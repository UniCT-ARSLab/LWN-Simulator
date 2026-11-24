import { PropsWithChildren } from "react";

/**
 * Properties for configuring a popup component.
 */
export interface PopupProps extends PropsWithChildren {
  /**
   * Additional CSS class names applied to the popup element.
   */
  className?: string;

  /**
   * State and actions used to control the popup behavior.
   */
  logic: PopupLogic;

  /**
   * Title displayed in the popup header.
   */
  title: string;

  /**
   * Whether the close button, Escape and clicks outside can dismiss the popup.
   * Defaults to true.
   */
  dismissible?: boolean;
}

/**
 * Properties accepted by the popup logic hook.
 *
 * The hook currently does not require configuration, but the named type keeps
 * the public API ready for future options.
 */
export type UsePopupProps = Record<string, never>;

/**
 * State and actions exposed by the popup logic hook.
 */
export interface PopupLogic {
  /**
   * Indicates whether the popup is currently open.
   *
   * Controls the popup state and user interaction availability.
   */
  isOpen: boolean;

  /**
   * Indicates whether the popup should remain rendered and visible.
   *
   * Can be used to handle visibility transitions or animations independently
   * from the open state.
   */
  isVisible: boolean;

  /**
   * Opens the popup.
   */
  openPopup: () => void;

  /**
   * Closes the popup.
   */
  closePopup: () => void;
}
