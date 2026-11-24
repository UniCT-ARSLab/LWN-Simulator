import { JSX } from "react/jsx-runtime";

/**
 * Defines the visual style variant of a button.
 */
export enum ButtonType {
  /** Main button style for primary actions. */
  Primary = "primary",

  /** Secondary button style for less prominent actions. */
  Secondary = "secondary",

  /** Button style with transparent background and visible border. */
  Outlined = "outlined",
}

/**
 * Defines the layout behavior of a button component.
 */
export enum ButtonLayout {
  /** Default button layout with standard dimensions. */
  Default = "default",

  /** Button expands to occupy the full available width of its container. */
  Block = "block",

  /** Button layout optimized for displaying only an icon. */
  Icon = "icon",
}

/**
 * Defines the font family used by the button content.
 */
export enum ButtonFont {
  /** Uses the primary font family. */
  Primary = "--primary-font",

  /** Uses the secondary font family. */
  Secondary = "--secondary-font",
}

/**
 * Properties for configuring a button component.
 */
export interface ButtonProps {
  /**
   * Content displayed inside the button.
   *
   * Supports both plain text values and custom React elements.
   */
  value: string | JSX.Element;

  /**
   * Additional CSS class names applied to the button element.
   */
  className?: string;

  /**
   * Visual style variant of the button.
   *
   * @default ButtonType.Primary
   */
  type?: ButtonType;

  /**
   * Font family used for the button content.
   *
   * @default ButtonFont.Primary
   */
  font?: ButtonFont;

  /**
   * Layout variant controlling the button dimensions and structure.
   *
   * @default ButtonLayout.Default
   */
  layout?: ButtonLayout;

  /**
   * Prevents user interaction and applies the disabled visual state.
   *
   * @default false
   */
  disabled?: boolean;

  /**
   * Displays the loading state while an external asynchronous operation is in progress.
   *
   * @default false
   */
  loading?: boolean;

  /**
   * Callback executed when the button is clicked.
   *
   * Supports both synchronous and asynchronous handlers.
   */
  onClick?: (e?: React.MouseEvent) => void | Promise<void>;
}
