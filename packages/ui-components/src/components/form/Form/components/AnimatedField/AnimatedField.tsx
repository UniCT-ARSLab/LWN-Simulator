"use client";

import { useEffect, useState } from "react";
import cn from "classnames";
import { AnimatedFieldProps } from "./AnimatedField.types";

export const AnimatedField = (props: AnimatedFieldProps) => {
  const [mounted, setMounted] = useState(props.visible);
  const [animation, setAnimation] = useState("");

  useEffect(() => {
    if (props.visible && !mounted) {
      setMounted(true);
      setAnimation("in");
    }

    if (!props.visible && mounted) {
      setAnimation("out");
    }
  }, [props.visible, mounted]);

  if (!mounted) return null;

  return (
    <div
      className={cn("form-field-wrapper", animation)}
      onAnimationEnd={() => {
        if (animation === "out") {
          setMounted(false);
          setAnimation("");
        }

        if (animation === "in") {
          setAnimation("");
        }
      }}
    >
      <div className="form-field-header">
        <label className="field-label" htmlFor={props.field.name}>
          {props.field.label}
          {props.field.required && "*"}
        </label>

        <div className="error">{props.field.error}</div>
      </div>

      <div className="field-wrapper">
        {props.field.render({
          ...props.field,
          value: props.formLogic.fieldsState[props.field.name].value,
          setValue: (v) => props.formLogic.setValue(props.field.name, v),
          disabled: props.formLogic.isFieldDisabled(
            props.field,
            props.formLogic.fieldsState,
          ),
        })}
        {props.field.toolbar && (
          <div className="field-toolbar">{props.field.toolbar}</div>
        )}
      </div>

      {props.field.info && (
        <div className="field-info">
          <span className="material-symbols-outlined icon">info</span>
          <span>
            {props.field.info[props.field.value as string] ||
              props.field.info.default}
          </span>
        </div>
      )}
    </div>
  );
};

export default AnimatedField;
