import { RefObject } from "react";

/**
 * Properties for configuring the outside click detection hook.
 *
 * The hook triggers a callback when a user interaction occurs outside
 * the referenced HTML element.
 *
 * @template T The HTML element type associated with the reference.
 */
export interface UseOutsideAlerterProps<T extends HTMLElement = HTMLElement> {
  /**
   * Reference to the target element used as the boundary for outside click detection.
   */
  ref: RefObject<T | null>;

  /**
   * Callback executed when a click occurs outside the referenced element.
   */
  onClickOutside: () => void;
}
