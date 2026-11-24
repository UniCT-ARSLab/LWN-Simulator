"use client";

import { SimulationEvent } from "@lwn-simulator/contracts";
import { CSSProperties, UIEvent, useEffect, useRef, useState } from "react";
import { VirtualEventListProps } from "./VirtualEventList.types";
import { formatRawRadioPayload } from "@/utils/radioFrame";

const eventRowHeight = 76;
const virtualOverscan = 5;

const formatEventType = (value: string): string =>
  value.replace(/-/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());

const formatSimulationTime = (milliseconds: number): string => {
  const totalSeconds = Math.max(0, milliseconds) / 1000;
  return `t+${totalSeconds.toFixed(totalSeconds >= 100 ? 0 : 1)}s`;
};

const formatDecodedPayload = (
  payload: SimulationEvent["decodedPayload"],
): string | null => {
  if (!payload) return null;
  try {
    return JSON.stringify(payload);
  } catch {
    return null;
  }
};

export const VirtualEventList = (props: VirtualEventListProps) => {
  // States
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(320);

  // Hooks
  const listRef = useRef<HTMLDivElement>(null);

  const firstVisible = Math.floor(scrollTop / eventRowHeight);
  const visibleCount = Math.ceil(viewportHeight / eventRowHeight);
  const start = Math.max(0, firstVisible - virtualOverscan);
  const end = Math.min(
    props.events.length,
    firstVisible + visibleCount + virtualOverscan,
  );

  // Effects
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const observer = new ResizeObserver(() =>
      setViewportHeight(list.clientHeight || 320),
    );
    observer.observe(list);
    setViewportHeight(list.clientHeight || 320);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const list = listRef.current;
    if (list && props.autoScroll) list.scrollTop = list.scrollHeight;
  }, [props.autoScroll, props.events.length]);

  if (props.events.length === 0) return <p>{props.emptyMessage}</p>;

  return (
    <div
      className="logs-page__virtual-list virtual-event-list"
      onScroll={(event: UIEvent<HTMLDivElement>) => {
        const list = event.currentTarget;
        const nearEnd =
          list.scrollHeight - list.scrollTop - list.clientHeight <= 96;
        setScrollTop(list.scrollTop);
        props.onNearEnd?.(nearEnd);
        if (list.scrollTop <= 96) props.onReachStart?.();
      }}
      ref={listRef}
    >
      <div
        style={{
          height: props.events.length * eventRowHeight,
          position: "relative",
        }}
      >
        {props.events.slice(start, end).map((event, offset) => {
          const index = start + offset;
          const decodedPayload = formatDecodedPayload(event.decodedPayload);
          const rawPayload = formatRawRadioPayload(event.radioFrame);
          const style: CSSProperties = {
            height: eventRowHeight,
            transform: `translateY(${index * eventRowHeight}px)`,
          };

          return (
            <div className="logs-page__event" key={event.id} style={style}>
              <time>
                #{event.sequence}
                <small>{formatSimulationTime(event.timestampMilliseconds)}</small>
              </time>
              <span>{formatEventType(event.type)}</span>
              <small title={[event.message, decodedPayload ? `Decoded ${decodedPayload}` : null, rawPayload, event.error ? `Error: ${event.error}` : null].filter(Boolean).join(" · ")}>
                {event.message}
                {decodedPayload ? ` · Decoded ${decodedPayload}` : ""}
                {rawPayload ? ` · ${rawPayload}` : ""}
                {event.error ? ` · Error: ${event.error}` : ""}
                {event.gatewayID ? ` · Gateway ${event.gatewayID}` : ""}
                {event.deviceID ? ` · Device ${event.deviceID}` : ""}
              </small>
            </div>
          );
        })}
      </div>
    </div>
  );
};
