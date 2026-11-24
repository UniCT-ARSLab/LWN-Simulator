/**
 * Defines the color variant of a logo component.
 */
export enum LogoType {
  /** Uses the primary theme color. */
  Primary = "--primary-color",

  /** Uses the secondary theme color. */
  Secondary = "--secondary-color",
}

/**
 * Defines the background style applied to a logo component.
 */
export enum LogoLayout {
  /** Displays the logo on a transparent background. */
  Default = "transparent",

  /** Displays the logo on the primary theme background. */
  PrimaryBackground = "var(--primary-bg-color)",

  /** Displays the logo on the secondary theme background. */
  SecondaryBackground = "var(--secondary-bg-color)",

  /** Displays the logo on the tertiary theme background. */
  TertiaryBackground = "var(--tertiary-bg-color)",
}

/**
 * Defines the available sizes for a logo component.
 */
export enum LogoSize {
  /** Small logo size. */
  Sm = "50px",

  /** Medium logo size. */
  Md = "70px",

  /** Large logo size. */
  Lg = "90px",
}

/**
 * Properties for configuring a logo component.
 */
export interface LogoProps {
  /**
   * Color variant applied to the logo.
   *
   * @default LogoType.Primary
   */
  type?: LogoType;

  /**
   * Background style used when rendering the logo.
   *
   * @default LogoLayout.Default
   */
  layout?: LogoLayout;

  /**
   * Size of the rendered logo.
   *
   * @default LogoSize.Md
   */
  size?: LogoSize;
}
