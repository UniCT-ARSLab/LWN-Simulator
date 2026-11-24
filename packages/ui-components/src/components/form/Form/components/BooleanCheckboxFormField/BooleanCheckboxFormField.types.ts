import { FormField, FormFieldProps } from "../..";

/**
 * Properties for configuring a boolean checkbox form field.
 *
 * Extends the base form field properties with single checkbox behavior.
 */
export interface BooleanCheckboxFormFieldProps extends FormFieldProps {
  /**
   * Current checked state of the checkbox.
   */
  value: boolean;

  /**
   * Updates the checked state of the checkbox.
   *
   * @param value New checked state.
   */
  setValue(value: boolean): void;

  /**
   * Text displayed alongside the checkbox.
   */
  text: string;

  /**
   * Boolean checkbox fields do not support placeholder text.
   */
  placeholder?: undefined;

  /**
   * Boolean checkbox fields do not support value formatting.
   */
  format?: undefined;

  /**
   * Boolean checkbox fields do not support custom validation rules.
   */
  validations?: undefined;
}

/**
 * Configuration options for creating a boolean checkbox form field.
 *
 * Excludes rendering logic and value update handling, which are managed
 * internally by the form system.
 *
 * The `disabled` property supports both static and dynamic behavior based
 * on the current form state.
 */
export interface BooleanCheckboxFieldOptions extends Omit<
  BooleanCheckboxFormFieldProps,
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
