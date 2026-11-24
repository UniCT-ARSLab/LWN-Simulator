import TextAreaFormField from "./TextAreaFormField";
import { FormField } from "../../Form.types";
import {
  TextAreaFieldOptions,
  TextAreaFormFieldProps,
} from "./TextAreaFormField.types";

export const textAreaField = (props: TextAreaFieldOptions): FormField => {
  return {
    ...props,

    render: (props: TextAreaFormFieldProps) => {
      return <TextAreaFormField {...props} />;
    },
  };
};
