"use client";

import { useState } from "react";
import {
  ButtonFont,
  ButtonLayout,
  ButtonProps,
  ButtonType,
} from "./Button.types";
import cn from "classnames";
import Spinner, { SpinnerSize, SpinnerType } from "../Spinner";

const Button = (props: ButtonProps) => {
  const type = props.type || ButtonType.Primary;
  const font = props.font || ButtonFont.Primary;
  const layout = props.layout || ButtonLayout.Default;

  //States
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const loading = props.loading || isLoading;

  return (
    <button
      onClick={async (e) => {
        setIsLoading(true);
        try {
          await props.onClick?.(e);
        } finally {
          setIsLoading(false);
        }
      }}
      disabled={props.disabled || loading}
      className={cn(
        "button",
        props.className,
        type,
        layout,
        props.disabled && "disabled",
        loading && "loading",
      )}
      style={{ fontFamily: `var(${font})` }}
    >
      {props.value}
      {loading && (
        <div className="loading-wrapper">
          <Spinner
            size={SpinnerSize.Sm}
            type={
              type === ButtonType.Primary
                ? SpinnerType.Secondary
                : SpinnerType.Primary
            }
          />
        </div>
      )}
    </button>
  );
};

export default Button;
