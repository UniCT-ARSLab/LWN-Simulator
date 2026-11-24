import { BooleanCheckboxFormFieldProps } from "./BooleanCheckboxFormField.types";
import cn from "classnames";

const BooleanCheckboxFormField = (props: BooleanCheckboxFormFieldProps) => {
  return (
    <div
      className={cn(
        "form-field boolean-checkbox-form-field grid",
        props.disabled && "disabled",
      )}
    >
      <div
        className="box"
        onClick={() => {
          if (props.disabled) return;

          props.setValue(!props.value);
        }}
      >
        <input
          className="checkbox-option"
          type="checkbox"
          id={props.name}
          checked={props.value}
          readOnly
          placeholder={props.placeholder}
          name={props.name}
          disabled={props.disabled}
        />
        <span className="material-symbols-outlined checkbox-icon">
          {props.value ? "check_box" : "check_box_outline_blank"}
        </span>
        <span>{props.text}</span>
      </div>
    </div>
  );
};

export default BooleanCheckboxFormField;
