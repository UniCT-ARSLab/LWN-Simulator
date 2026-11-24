import RadioFormField from "./RadioFormField";
import { FormField } from "../../Form.types";
import { RadioFieldOptions, RadioFormFieldProps } from "./RadioFormField.types";

export const radioField = (props: RadioFieldOptions): FormField => {
  return {
    ...props,

    render: (props: RadioFormFieldProps) => {
      return <RadioFormField {...props} />;
    },
  };
};
