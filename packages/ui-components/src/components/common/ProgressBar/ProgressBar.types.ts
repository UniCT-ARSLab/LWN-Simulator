/**
 * Defines the available color variants for a progress bar component.
 */
export enum ProgressBarType {
  /** Uses the primary theme color. */
  Primary = "--primary-color",

  /** Uses the secondary theme color. */
  Secondary = "--secondary-color",
}

/**
 * Properties for configuring a progress bar component.
 */
export interface ProgressBarProps {
  /**
   * Completion percentage represented by the progress bar.
   *
   * The value should be expressed as a number between `0` and `100`.
   */
  percentage: number;

  /**
   * Color variant applied to the progress indicator.
   *
   * @default ProgressBarType.Primary
   */
  type?: ProgressBarType;
}
