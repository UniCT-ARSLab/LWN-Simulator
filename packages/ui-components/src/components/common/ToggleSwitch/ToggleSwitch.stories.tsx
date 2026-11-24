import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import ToggleSwitch from "./ToggleSwitch";
import { useState } from "react";
import { ToggleSwitchProps, ToggleSwitchType } from "./ToggleSwitch.types";

const meta = {
  title: "ui-components/common/ToggleSwitch",
  component: ToggleSwitch,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof ToggleSwitch>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleToggleSwitch: Story = {
  args: {} as ToggleSwitchProps,
  render: () => {
    // States
    const [isActive, setIsActive] = useState<boolean>(false);

    return (
      <ToggleSwitch
        value={isActive}
        onToggle={(active) => {
          setIsActive(active);
        }}
      />
    );
  },
};

export const ChooseToggleSwitch: Story = {
  args: { type: ToggleSwitchType.Choose } as ToggleSwitchProps,
  render: (args) => {
    // States
    const [isActive, setIsActive] = useState<boolean>(false);

    return (
      <ToggleSwitch
        {...args}
        value={isActive}
        onToggle={(active) => {
          setIsActive(active);
        }}
      />
    );
  },
};

export const IconToggleSwitch: Story = {
  args: {
    trueIcon: <span className="material-symbols-outlined">check</span>,
    falseIcon: <span className="material-symbols-outlined">close_small</span>,
  } as ToggleSwitchProps,
  render: (args) => {
    // States
    const [isActive, setIsActive] = useState<boolean>(false);

    return (
      <ToggleSwitch
        {...args}
        value={isActive}
        onToggle={(active) => {
          setIsActive(active);
        }}
      />
    );
  },
};

export const IconChooseToggleSwitch: Story = {
  args: {
    type: ToggleSwitchType.Choose,
    trueIcon: <span className="material-symbols-outlined">bedtime</span>,
    falseIcon: <span className="material-symbols-outlined">sunny</span>,
  } as ToggleSwitchProps,
  render: (args) => {
    // States
    const [isActive, setIsActive] = useState<boolean>(false);

    return (
      <ToggleSwitch
        {...args}
        value={isActive}
        onToggle={(active) => {
          setIsActive(active);
        }}
      />
    );
  },
};
