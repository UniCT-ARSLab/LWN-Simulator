"use client";

import { CSSProperties, useEffect, useMemo, useRef, useState } from "react";
import { SelectFormFieldProps } from "./SelectFormField.types";
import cn from "classnames";
import { useOutsideAlerter } from "../../../../../hooks";

const virtualOptionStride = 44;
const virtualOptionOverscan = 3;
const virtualOptionViewport = virtualOptionStride * 5;
const virtualizeAfterOptions = 100;

const SelectFormField = (props: SelectFormFieldProps) => {
  const { setValue, value } = props;

  // States
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [choicesScrollTop, setChoicesScrollTop] = useState(0);
  const selectRef = useRef<HTMLDivElement | null>(null);
  const choicesRef = useRef<HTMLDivElement | null>(null);

  useOutsideAlerter({
    ref: selectRef,
    onClickOutside: () => setIsOpen(false),
  });

  // Effects
  useEffect(() => {
    if (props.disabled) return;

    if (value === "") setValue(null);
  }, [props.disabled, setValue, value]);

  const shouldVirtualize = props.options.length > virtualizeAfterOptions;

  useEffect(() => {
    if (!isOpen || !shouldVirtualize) return;

    const selectedIndex = props.options.findIndex(
      (choice) => choice.value === props.value,
    );
    const nextScrollTop = Math.max(
      0,
      selectedIndex * virtualOptionStride - virtualOptionViewport / 2,
    );

    if (choicesRef.current) choicesRef.current.scrollTop = nextScrollTop;
    setChoicesScrollTop(nextScrollTop);
  }, [isOpen, props.options, props.value, shouldVirtualize]);

  // Memos
  const displayed = useMemo(() => {
    const choice = props.options.find((c) => c.value === props.value);
    return choice?.displayed ?? <>{choice?.value}</>;
  }, [props.options, props.value]);
  const visibleChoiceRange = useMemo(() => {
    if (!shouldVirtualize) {
      return { start: 0, end: props.options.length };
    }

    const firstVisible = Math.floor(choicesScrollTop / virtualOptionStride);
    const visibleCount = Math.ceil(
      virtualOptionViewport / virtualOptionStride,
    );

    return {
      start: Math.max(0, firstVisible - virtualOptionOverscan),
      end: Math.min(
        props.options.length,
        firstVisible + visibleCount + virtualOptionOverscan,
      ),
    };
  }, [choicesScrollTop, props.options.length, shouldVirtualize]);
  const visibleChoices = props.options.slice(
    visibleChoiceRange.start,
    visibleChoiceRange.end,
  );

  return (
    <div
      ref={selectRef}
      className={cn(
        "form-field select-form-field",
        isOpen && "open",
        props.disabled && "disabled",
      )}
    >
      <div
        className={cn(
          "selected-box",
          (props.value === null || props.value === "") && "placeholder",
        )}
        onMouseDown={(e) => {
          e.preventDefault();
        }}
        onClick={(e) => {
          e.preventDefault();
          if (props.disabled) return;

          setIsOpen((prev) => !prev);
        }}
      >
        {props.value !== null && props.value !== ""
          ? displayed
          : props.placeholder}
      </div>
      <span className="material-symbols-outlined arrow-icon">
        keyboard_arrow_down
      </span>
      {isOpen && (
        <div
          ref={choicesRef}
          className={cn("choices", shouldVirtualize && "virtualized")}
          style={{ "--options": props.options.length } as CSSProperties}
          onScroll={(event) => {
            if (shouldVirtualize) {
              setChoicesScrollTop(event.currentTarget.scrollTop);
            }
          }}
        >
          {shouldVirtualize && (
            <div
              className="choices-space"
              style={{ height: props.options.length * virtualOptionStride }}
            />
          )}
          {visibleChoices.map((choice, offset) => {
            const optionIndex = visibleChoiceRange.start + offset;

            return (
              <div
                className={cn(
                  "choice",
                  choice.value == props.value && "selected",
                )}
                onClick={() => {
                  if (props.disabled) return;

                  props.setValue(choice.value);
                  setIsOpen(false);
                }}
                key={choice.value}
                style={
                  shouldVirtualize
                    ? {
                        transform: `translateY(${optionIndex * virtualOptionStride}px)`,
                      }
                    : undefined
                }
              >
                {choice.displayed || choice.value}
                {choice.value == props.value && (
                  <span className="material-symbols-outlined check-icon">
                    check
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default SelectFormField;
