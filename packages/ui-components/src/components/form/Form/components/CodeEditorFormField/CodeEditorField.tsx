import CodeEditorFormField from "./CodeEditorFormField";
import { FormField } from "../../Form.types";
import {
  CodeEditorFieldOptions,
  CodeEditorFormFieldProps,
} from "./CodeEditorFormField.types";

/** Creates a multiline code editor field for the shared form component. */
export const codeEditorField = (
  props: CodeEditorFieldOptions,
): FormField => ({
  ...props,
  render: (fieldProps: CodeEditorFormFieldProps) => (
    <CodeEditorFormField {...fieldProps} />
  ),
});
