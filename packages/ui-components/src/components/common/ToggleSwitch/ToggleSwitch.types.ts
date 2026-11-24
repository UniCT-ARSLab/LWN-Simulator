import { JSX } from "react/jsx-runtime";

/**
 * Defines the available interaction styles for a toggle switch component.
 */
export enum ToggleSwitchType {
  /** Standard on/off toggle behavior. */
  Boolean = "boolean",

  /** Toggle used to choose between two alternatives. */
  Choose = "choose",
}

/**
 * Properties for configuring a toggle switch component.
 */
export interface ToggleSwitchProps {
  /**
   * Current state of the toggle switch.
   */
  value: boolean;

  /**
   * Callback executed when the toggle state changes.
   *
   * @param active The new state of the toggle switch.
   */
  onToggle: (active: boolean) => void;

  /**
   * Interaction style of the toggle switch.
   *
   * @default ToggleSwitchType.Boolean
   */
  type?: ToggleSwitchType;

  /**
   * Custom icon displayed when the toggle is in the active state.
   */
  trueIcon?: JSX.Element;

  /**
   * Custom icon displayed when the toggle is in the inactive state.
   */
  falseIcon?: JSX.Element;
}
