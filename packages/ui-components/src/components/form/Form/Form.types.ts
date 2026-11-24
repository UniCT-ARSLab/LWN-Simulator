import { ButtonProps } from "@/components/common";
import { JSX, ReactNode } from "react";

/**
 * Represents the possible values supported by form fields.
 */
export type FormValue = string | number | boolean | Date | null | Array<string>;

/**
 * Properties passed to a custom form field renderer.
 *
 * Extends the base field configuration while providing access to the
 * value update callback and field state information.
 */
export interface FormFieldProps extends Omit<FormField, "render"> {
  /**
   * Updates the value of the current field.
   *
   * @param value New value assigned to the field.
   */
  setValue(value: FormValue): void;

  /**
   * Indicates whether the field is disabled.
   *
   * Disabled fields should prevent user interaction and value changes.
   */
  disabled?: boolean;

  /** Prevents editing while keeping the field visually enabled. */
  readOnly?: boolean;

  /** Maximum number of characters accepted by multiline text fields. */
  charsMax?: number;
}

/**
 * Defines a validation rule applied to a form field.
 */
export interface FormValidation {
  /**
   * Rule used to validate the field value. Predicate rules are useful for
   * structured values whose validity cannot be expressed reliably with a
   * regular expression.
   */
  rule: RegExp | ((value: FormValue) => boolean);

  /**
   * Error message displayed when validation fails.
   */
  error: string;

  /**
   * Determines whether the validation is executed while the user is typing.
   *
   * When disabled, validation is performed only during form submission
   * or explicit validation calls.
   */
  realtime?: boolean;
}

/**
 * Configuration of a single form field.
 */
export interface FormField {
  /**
   * Unique identifier of the field.
   */
  name: string;

  /**
   * Label displayed for the field.
   */
  label: string;

  /**
   * Current value assigned to the field.
   */
  value: FormValue;

  /**
   * Callback invoked when the field value changes.
   *
   * Receives the current form logic, which can be used to access the form
   * state or interact with other fields.
   */
  onChange?: (logic: FormLogic) => void;

  /**
   * Placeholder text displayed when the field has no value.
   */
  placeholder?: string;

  /**
   * Additional information or helper text displayed alongside the field.
   *
   * The key is matched against the field value. If no key matches,
   * the `default` value is used.
   */
  info?: {
    default: string;
    [value: string]: string;
  };

  /**
   * Function used to transform the raw input value before storing it.
   *
   * @param raw Raw string value entered by the user.
   * @returns Formatted value stored in the field state.
   */
  format?: (raw: string) => string;

  /**
   * Collection of validation rules applied to the field.
   */
  validations?: Array<FormValidation>;

  /**
   * Current validation error associated with the field.
   *
   * Contains `null` when the field is valid.
   */
  error: string | null;

  /**
   * Indicates whether the field is mandatory.
   *
   * Required fields must contain a valid value before form submission.
   */
  required?: boolean;

  /**
   * Determines whether the field is disabled.
   *
   * Can be a static value or a function evaluated dynamically from the
   * current form state.
   */
  disabled?: boolean | ((fieldsState: Record<string, FormField>) => boolean);

  /** Prevents editing while keeping the field visually enabled. */
  readOnly?: boolean;

  /** Maximum number of characters accepted by multiline text fields. */
  charsMax?: number;

  /**
   * Determines whether the field is rendered.
   *
   * Can be a static value or a function evaluated dynamically from the
   * current form state.
   */
  display?: boolean | ((fieldsState: Record<string, FormField>) => boolean);

  /**
   * Optional custom content displayed alongside the field.
   *
   * Typically used to render additional controls or actions related to the
   * field.
   */
  toolbar?: JSX.Element;

  /**
   * Custom renderer used to define the field UI.
   *
   * Receives the field properties and the callback required to update
   * its value.
   */
  render(props: FormFieldProps): ReactNode;
}

/**
 * Properties for configuring a form component.
 */
export interface FormProps {
  /**
   * Collection of fields rendered by the form.
   */
  fields: Array<FormField>;

  /**
   * Callback executed when the form is submitted.
   *
   * Receives an object containing the current values indexed by field name.
   *
   * Supports both synchronous and asynchronous handlers.
   */
  onSubmit(values: Record<string, FormValue>): void | Promise<void>;

  /**
   * Configuration of the submit button displayed by the form.
   */
  submitButton?: ButtonProps;

  /**
   * Controls whether the form renders its submit button.
   *
   * @default true
   */
  showSubmitButton?: boolean;

  /**
   * Callback invoked when the form logic is initialized and becomes available.
   *
   * The provided logic can be used to access the form state and interact with
   * its fields programmatically.
   *
   * @param logic The form logic instance exposed by the form.
   */
  onLogicReady?: (logic: FormLogic) => void;
}

/**
 * Properties accepted by the form logic hook.
 *
 * Extends the form configuration properties.
 */
export interface UseFormProps extends FormProps {}

/**
 * State and actions exposed by the form logic hook.
 */
export interface FormLogic {
  /**
   * Current state of all form fields indexed by field name.
   */
  fieldsState: Record<string, FormField>;

  /**
   * Updates the value of a specific field.
   *
   * @param name Name of the field to update.
   * @param value New value assigned to the field.
   */
  setValue(name: string, value: FormValue): void;

  /**
   * Updates a property of a specific form field.
   *
   * @param name Name of the field to update.
   * @param prop Name of the property to update.
   * @param value New value assigned to the property.
   */
  setProp(name: string, prop: string, value: unknown): void;

  /**
   * Determines whether a field should be disabled.
   *
   * Evaluates the field configuration against the current form state.
   */
  isFieldDisabled: (
    field: FormField,
    fieldsState: Record<string, FormField>,
  ) => boolean;

  /**
   * Determines whether a field should be displayed.
   *
   * Evaluates the field visibility configuration against the current form state.
   */
  isFieldDisplayed: (
    field: FormField,
    fieldsState: Record<string, FormField>,
  ) => boolean;

  /**
   * Validates all form fields.
   *
   * @returns `true` when all fields pass validation, otherwise `false`.
   */
  validate: () => boolean;
}
