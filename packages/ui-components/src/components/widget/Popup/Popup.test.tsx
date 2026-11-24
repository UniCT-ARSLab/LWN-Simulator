import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { usePopup } from "./usePopup";

describe("usePopup", () => {
  afterEach(() => vi.useRealTimers());

  it("opens immediately and removes the popup after the close transition", () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => usePopup({}));

    act(() => result.current.openPopup());
    expect(result.current.isOpen).toBe(true);
    expect(result.current.isVisible).toBe(true);

    act(() => result.current.closePopup());
    expect(result.current.isVisible).toBe(false);
    expect(result.current.isOpen).toBe(true);

    act(() => vi.advanceTimersByTime(300));
    expect(result.current.isOpen).toBe(false);
  });
});
