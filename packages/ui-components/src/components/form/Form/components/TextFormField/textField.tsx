import TextFormField from "./TextFormField";
import { FormField } from "../../Form.types";
import { TextFieldOptions, TextFormFieldProps } from "./TextFormField.types";

export const textField = (props: TextFieldOptions): FormField => {
  return {
    ...props,

    render: (props: TextFormFieldProps) => {
      return <TextFormField {...props} />;
    },
  };
};
