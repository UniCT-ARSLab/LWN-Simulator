import { FormField, FormFieldProps } from "../../Form.types";

/**
 * Properties for configuring a text-based form field.
 *
 * Extends the base form field properties with text-specific behavior.
 */
export interface TextFormFieldProps extends FormFieldProps {
  /**
   * Current text value of the field.
   */
  value: string;

  /**
   * Updates the current text value.
   *
   * @param value New text value assigned to the field.
   */
  setValue(value: string): void;

  /**
   * Indicates whether the field value should be displayed using a mask.
   *
   * Useful for hiding sensitive text values such as passwords.
   *
   * @default false
   */
  masked?: boolean;
}

/**
 * Configuration options for creating a text form field.
 *
 * Excludes rendering logic and value update handling, which are managed
 * internally by the form system.
 *
 * The `disabled` property supports both static and dynamic behavior based
 * on the current form state.
 */
export interface TextFieldOptions extends Omit<
  TextFormFieldProps,
  "render" | "setValue" | "disabled"
> {
  /**
   * Determines whether the field is disabled.
   *
   * Can be a static value or a function evaluated dynamically from the
   * current form state.
   */
  disabled?: boolean | ((fieldsState: Record<string, FormField>) => boolean);
}
