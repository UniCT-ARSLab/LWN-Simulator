import { FormField, FormFieldProps } from "../../Form.types";

/** Properties used to render a range form field. */
export interface RangeFormFieldProps extends FormFieldProps {
  /** Current range value as a string. */
  value: string;
  /** Stores the updated range value. */
  setValue(value: string): void;
  /** Minimum accepted value. */
  min: number;
  /** Maximum accepted value. */
  max: number;
  /** Increment applied when changing the range. */
  step?: number;
  /** Formats the visible value shown beside the control. */
  formatValue?: (value: string) => string;
}

/** Options used to create a range form field descriptor. */
export interface RangeFieldOptions extends Omit<
  RangeFormFieldProps,
  "render" | "setValue" | "disabled"
> {
  disabled?: boolean | ((fieldsState: Record<string, FormField>) => boolean);
}
