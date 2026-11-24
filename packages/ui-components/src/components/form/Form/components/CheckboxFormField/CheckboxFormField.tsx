"use client";

import { useEffect } from "react";
import { CheckboxFormFieldProps } from "./CheckboxFormField.types";
import cn from "classnames";

const CheckboxFormField = (props: CheckboxFormFieldProps) => {
  const { disabled, setValue, value } = props;

  // Effects
  useEffect(() => {
    if (disabled) return;

    if (Array.isArray(value) && value.length === 0) setValue(null);
  }, [disabled, setValue, value]);

  return (
    <div
      className={cn(
        "form-field checkbox-form-field grid",
        props.disabled && "disabled",
      )}
    >
      {props.options.map((opt, i) => {
        return (
          <div
            className="choice"
            key={i}
            onClick={() => {
              if (props.disabled) return;

              if (!props.value) props.setValue([opt.value]);
              else if (!props.value.includes(opt.value)) {
                props.setValue([...props.value, opt.value]);
              } else {
                const tmp = [...props.value];
                tmp.splice(tmp.indexOf(opt.value), 1);

                props.setValue(tmp);
              }
            }}
          >
            <input
              className="checkbox-option"
              type="checkbox"
              id={`${props.name}-${opt.value}`}
              checked={
                Array.isArray(props.value) && props.value.includes(opt.value)
              }
              readOnly
              placeholder={props.placeholder}
              name={props.name}
              disabled={props.disabled}
            />
            <span className="material-symbols-outlined checkbox-icon">
              {Array.isArray(props.value) && props.value.includes(opt.value)
                ? "check_box"
                : "check_box_outline_blank"}
            </span>
            <span>{opt.displayed || opt.value}</span>
          </div>
        );
      })}
    </div>
  );
};

export default CheckboxFormField;
