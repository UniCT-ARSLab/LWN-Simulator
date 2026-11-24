import SelectFormField from "./SelectFormField";
import { FormField } from "../../Form.types";
import {
  SelectFieldOptions,
  SelectFormFieldProps,
} from "./SelectFormField.types";

export const selectField = (props: SelectFieldOptions): FormField => {
  return {
    ...props,

    render: (props: SelectFormFieldProps) => {
      return <SelectFormField {...props} />;
    },
  };
};
