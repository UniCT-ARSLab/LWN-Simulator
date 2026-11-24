/**
 * Defines the available sizes for a spinner component.
 */
export enum SpinnerSize {
  /** Small spinner size. */
  Sm = "0.4px",

  /** Medium spinner size. */
  Md = "0.8px",

  /** Large spinner size. */
  Lg = "1.2px",
}

/**
 * Defines the color variant of a spinner component.
 */
export enum SpinnerType {
  /** Uses the primary theme color. */
  Primary = "--primary-color",

  /** Uses the secondary theme color. */
  Secondary = "--secondary-color",
}

/**
 * Properties for configuring a spinner component.
 */
export interface SpinnerProps {
  /**
   * Size of the spinner indicator.
   *
   * @default SpinnerSize.Md
   */
  size?: SpinnerSize;

  /**
   * Color variant applied to the spinner.
   *
   * @default SpinnerType.Primary
   */
  type?: SpinnerType;
}
