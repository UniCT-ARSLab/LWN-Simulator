"use client";

import { UIEvent, useMemo, useState } from "react";
import { SimulationRuntimeListProps } from "./SimulationRuntimeList.types";

const runtimeRowHeight = 52;
const runtimeViewportHeight = 320;
const runtimeOverscan = 4;

export const SimulationRuntimeList = <T extends { id: string }>(
  props: SimulationRuntimeListProps<T>,
) => {
  // States
  const [scrollTop, setScrollTop] = useState(0);

  // Memos
  const range = useMemo(() => {
    const firstVisible = Math.floor(scrollTop / runtimeRowHeight);
    const visibleCount = Math.ceil(
      runtimeViewportHeight / runtimeRowHeight,
    );

    return {
      start: Math.max(0, firstVisible - runtimeOverscan),
      end: Math.min(
        props.items.length,
        firstVisible + visibleCount + runtimeOverscan,
      ),
    };
  }, [props.items.length, scrollTop]);

  if (props.items.length === 0) {
    return <p className="simulation-dashboard__empty">{props.emptyMessage}</p>;
  }

  return (
    <div
      className="simulation-dashboard__runtime-list simulation-runtime-list"
      onScroll={(event: UIEvent<HTMLDivElement>) =>
        setScrollTop(event.currentTarget.scrollTop)
      }
    >
      <div
        className="simulation-dashboard__runtime-list-space"
        style={{ height: props.items.length * runtimeRowHeight }}
      >
        {props.items.slice(range.start, range.end).map((item, offset) => {
          const index = range.start + offset;

          return (
            <div
              className="simulation-dashboard__runtime-row"
              key={item.id}
              style={{ transform: `translateY(${index * runtimeRowHeight}px)` }}
            >
              {props.renderItem(item)}
            </div>
          );
        })}
      </div>
    </div>
  );
};
