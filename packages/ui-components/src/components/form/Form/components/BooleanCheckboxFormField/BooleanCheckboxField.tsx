import BooleanCheckboxFormField from "./BooleanCheckboxFormField";
import { FormField } from "../../Form.types";
import {
  BooleanCheckboxFieldOptions,
  BooleanCheckboxFormFieldProps,
} from "./BooleanCheckboxFormField.types";

export const booleanCheckboxField = (
  props: BooleanCheckboxFieldOptions,
): FormField => {
  return {
    ...props,

    render: (props: BooleanCheckboxFormFieldProps) => {
      return <BooleanCheckboxFormField {...props} />;
    },
  };
};
