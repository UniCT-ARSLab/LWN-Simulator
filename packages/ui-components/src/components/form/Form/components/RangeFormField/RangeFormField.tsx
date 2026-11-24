import cn from "classnames";
import { RangeFormFieldProps } from "./RangeFormField.types";

const RangeFormField = (props: RangeFormFieldProps) => {
  return (
    <div className="range-form-field-control">
      <input
        aria-label={props.label}
        className={cn("form-field range-form-field", props.disabled && "disabled")}
        id={props.name}
        type="range"
        min={props.min}
        max={props.max}
        step={props.step}
        value={props.value}
        name={props.name}
        disabled={props.disabled}
        onChange={(event) => props.setValue(event.target.value)}
      />
      <output className="range-form-field-value">
        {props.formatValue?.(props.value) ?? props.value}
      </output>
    </div>
  );
};

export default RangeFormField;
