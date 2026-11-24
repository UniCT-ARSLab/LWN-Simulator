import { FormField } from "../../Form.types";
import RangeFormField from "./RangeFormField";
import { RangeFieldOptions, RangeFormFieldProps } from "./RangeFormField.types";

export const rangeField = (props: RangeFieldOptions): FormField => ({
  ...props,
  render: (fieldProps: RangeFormFieldProps) => (
    <RangeFormField {...fieldProps} />
  ),
});
