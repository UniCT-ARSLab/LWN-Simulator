import { JSX } from "react/jsx-runtime";
import { FormField, FormFieldProps } from "../..";

/**
 * Represents an option available in a select form field.
 */
export interface SelectOption {
  /**
   * Value assigned to the option.
   */
  value: string | number;

  /**
   * Custom content displayed instead of the default option value.
   */
  displayed?: JSX.Element;
}

/**
 * Properties for configuring a select-based form field.
 *
 * Extends the base form field properties with selection-specific behavior.
 */
export interface SelectFormFieldProps extends FormFieldProps {
  /**
   * Currently selected option value.
   *
   * A `null` value indicates that no option is currently selected.
   */
  value: string | number | null;

  /**
   * Updates the selected option value.
   *
   * @param value New selected value, or `null` to clear the selection.
   */
  setValue(value: string | number | null): void;

  /**
   * Available options displayed by the select field.
   */
  options: Array<SelectOption>;

  /**
   * Select fields do not support value formatting.
   */
  format?: undefined;

  /**
   * Select fields do not support custom validation rules.
   */
  validations?: undefined;
}

/**
 * Configuration options for creating a select form field.
 *
 * Excludes rendering logic and value update handling, which are managed
 * internally by the form system.
 *
 * The `disabled` property supports both static and dynamic behavior based
 * on the current form state.
 */
export interface SelectFieldOptions extends Omit<
  SelectFormFieldProps,
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
