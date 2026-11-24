"use client";

import { useEffect, useState } from "react";
import { VirtualObjectListProps } from "./VirtualObjectList.types";

const objectRowHeight = 76;
const virtualOverscan = 5;

const formatEventType = (value: string): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

export const VirtualObjectList = (props: VirtualObjectListProps) => {
  // States
  const [scrollTop, setScrollTop] = useState(0);

  const firstVisible = Math.floor(scrollTop / objectRowHeight);
  const start = Math.max(0, firstVisible - virtualOverscan);
  const end = Math.min(
    props.objects.length,
    firstVisible + Math.ceil(320 / objectRowHeight) + virtualOverscan,
  );

  // Effects
  useEffect(() => setScrollTop(0), [props.objects]);

  if (props.objects.length === 0) return <p>No matching devices or gateways.</p>;

  return (
    <div
      className="logs-page__objects virtual-object-list"
      onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}
    >
      <div
        style={{
          height: props.objects.length * objectRowHeight,
          position: "relative",
        }}
      >
        {props.objects.slice(start, end).map((object, offset) => {
          const index = start + offset;

          return (
            <button
              className={
                props.selectedObject?.id === object.id
                  ? "logs-page__object logs-page__object--active"
                  : "logs-page__object"
              }
              key={object.id}
              onClick={() => props.onSelect(object)}
              style={{ transform: `translateY(${index * objectRowHeight}px)` }}
              type="button"
            >
              <span>{object.name}</span>
              <small>
                {formatEventType(object.kind)} · {object.identifier}
              </small>
            </button>
          );
        })}
      </div>
    </div>
  );
};
