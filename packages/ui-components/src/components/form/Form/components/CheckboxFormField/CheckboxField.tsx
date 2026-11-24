import CheckboxFormField from "./CheckboxFormField";
import { FormField } from "../../Form.types";
import {
  CheckboxFieldOptions,
  CheckboxFormFieldProps,
} from "./CheckboxFormField.types";

export const checkboxField = (props: CheckboxFieldOptions): FormField => {
  return {
    ...props,

    render: (props: CheckboxFormFieldProps) => {
      return <CheckboxFormField {...props} />;
    },
  };
};
