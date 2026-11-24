import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import RangeFormField from "./RangeFormField";
import { RangeFormFieldProps } from "./RangeFormField.types";

const meta = {
  title: "ui-components/form/fields/RangeFormField",
  component: RangeFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof RangeFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulRange = (props: { initialValue: string; disabled?: boolean }) => {
  const [value, setValue] = useState(props.initialValue);

  return (
    <RangeFormField
      name="speed"
      label="Speed"
      value={value}
      error={null}
      min={0.1}
      max={4}
      step={0.1}
      formatValue={(currentValue) => `${currentValue}x`}
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const Adjustable: Story = {
  args: {} as RangeFormFieldProps,
  render: () => <StatefulRange initialValue="1" />,
};

export const Disabled: Story = {
  args: {} as RangeFormFieldProps,
  render: () => <StatefulRange initialValue="2" disabled />,
};
