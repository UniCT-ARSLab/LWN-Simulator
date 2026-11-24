import { FormField, FormFieldProps } from "../..";

/**
 * Properties for configuring a textarea-based form field.
 *
 * Extends the base form field properties with multiline text-specific behavior.
 */
export interface TextAreaFormFieldProps extends FormFieldProps {
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
   * Enables resizing of the textarea element.
   *
   * When enabled, the user can manually adjust the textarea height.
   *
   * @default false
   */
  resize?: boolean;

  /**
   * Maximum number of characters allowed in the field.
   *
   * When provided, input exceeding this limit is prevented.
   */
  charsMax?: number;
}

/**
 * Configuration options for creating a textarea form field.
 *
 * Excludes rendering logic and value update handling, which are managed
 * internally by the form system.
 *
 * The `disabled` property supports both static and dynamic behavior based
 * on the current form state.
 */
export interface TextAreaFieldOptions extends Omit<
  TextAreaFormFieldProps,
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
