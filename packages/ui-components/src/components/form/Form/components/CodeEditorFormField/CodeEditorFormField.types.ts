import { FormField, FormFieldProps } from "../..";

/**
 * Properties rendered by the multiline code editor form field.
 *
 * The field intentionally keeps the value as plain text. Syntax validation
 * and execution are responsibilities of the consuming feature.
 */
export interface CodeEditorFormFieldProps extends FormFieldProps {
  /** Current source code value. */
  value: string;

  /** Updates the source code value. */
  setValue(value: string): void;

  /** Allows the editor to be resized vertically by the user. */
  resize?: boolean;
}

/**
 * Configuration options used to create a code editor field for `Form`.
 */
export interface CodeEditorFieldOptions extends Omit<
  CodeEditorFormFieldProps,
  "render" | "setValue" | "disabled"
> {
  /** Static or form-state-dependent disabled state. */
  disabled?: boolean | ((fieldsState: Record<string, FormField>) => boolean);
}
