import cn from "classnames";
import { TextFormFieldProps } from "./TextFormField.types";

const TextFormField = (props: TextFormFieldProps) => {
  return (
    <input
      className={cn("form-field text-form-field", props.disabled && "disabled")}
      id={props.name}
      type={props.masked ? "password" : "text"}
      value={props.value}
      name={props.name}
      spellCheck={false}
      disabled={props.disabled}
      readOnly={props.readOnly}
      onChange={(e) => {
        let value = e.target.value;

        if (props.format) value = props.format(value);

        props.setValue(value);
      }}
      placeholder={props.placeholder}
    />
  );
};

export default TextFormField;
