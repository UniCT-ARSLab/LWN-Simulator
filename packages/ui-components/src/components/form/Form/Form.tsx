"use client";

import { Button } from "@/components/common";
import { FormProps } from "./Form.types";
import { useForm } from "./useForm";
import cn from "classnames";
import { useEffect, useMemo, useRef } from "react";
import { AnimatedField } from "./components";

const Form = (props: FormProps) => {
  const formLogic = useForm({ ...props });
  const onLogicReady = props.onLogicReady;

  // Memos
  const visibleFields = useMemo(
    () =>
      Object.values(formLogic.fieldsState).filter((field) => {
        if (field.display === undefined) return true;

        return typeof field.display === "function"
          ? field.display(formLogic.fieldsState)
          : field.display;
      }),
    [formLogic.fieldsState],
  );

  // Hooks
  const mounted = useRef(false);

  // Effects
  useEffect(() => {
    onLogicReady?.(formLogic);
  }, [formLogic, onLogicReady]);

  useEffect(() => {
    mounted.current = true;
  }, []);

  return (
    <form className="form" noValidate onSubmit={(e) => e.preventDefault()}>
      {Object.values(formLogic.fieldsState).map((field) => (
        <AnimatedField
          key={field.name}
          field={field}
          visible={visibleFields.some((visible) => visible.name === field.name)}
          formLogic={formLogic}
        />
      ))}

      {props.showSubmitButton !== false && (
        <Button
          {...(props.submitButton || { value: "Submit" })}
          className={cn("form-button", props.submitButton?.className)}
          onClick={(e) => {
            e?.preventDefault();

            const isValid = formLogic.validate();

            if (isValid) {
              props.onSubmit(
                Object.fromEntries(
                  Object.values(formLogic.fieldsState)
                    .filter(
                      (field) =>
                        !formLogic.isFieldDisabled(
                          field,
                          formLogic.fieldsState,
                        ) &&
                        formLogic.isFieldDisplayed(field, formLogic.fieldsState),
                    )
                    .map((field) => [field.name, field.value]),
                ),
              );
            }
          }}
          disabled={
            props.submitButton?.disabled ||
            !!Object.values(formLogic.fieldsState).find(
              (field) =>
                !formLogic.isFieldDisabled(field, formLogic.fieldsState) &&
                field.error,
            )
          }
        />
      )}
    </form>
  );
};

export default Form;
