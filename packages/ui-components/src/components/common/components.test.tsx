import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import Button from "./Button/Button";
import ProgressBar from "./ProgressBar/ProgressBar";
import ToggleSwitch from "./ToggleSwitch/ToggleSwitch";

describe("common UI components", () => {
  it("calls the button handler and exposes loading and disabled states", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn().mockResolvedValue(undefined);
    const { rerender } = render(
      <Button value="Save" onClick={onClick} loading />,
    );
    const button = screen.getByRole("button", { name: "Save" });

    expect(button).toBeDisabled();
    expect(button).toHaveClass("loading");

    rerender(<Button value="Save" onClick={onClick} />);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("toggles the active state through its callback", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    const { container } = render(
      <ToggleSwitch value={false} onToggle={onToggle} />,
    );
    const toggle = container.querySelector(".toggle-switch");

    expect(toggle).not.toBeNull();
    await user.click(toggle as Element);

    expect(onToggle).toHaveBeenCalledWith(true);
  });

  it("renders the requested progress percentage", () => {
    const { container } = render(<ProgressBar percentage={42.5} />);

    expect(container.querySelector(".progress")).toHaveStyle({
      width: "42.5%",
    });
  });
});
