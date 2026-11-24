import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import BooleanCheckboxFormField from "./BooleanCheckboxFormField";
import { BooleanCheckboxFormFieldProps } from "./BooleanCheckboxFormField.types";

const meta = {
  title: "ui-components/form/fields/BooleanCheckboxFormField",
  component: BooleanCheckboxFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof BooleanCheckboxFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulCheckbox = (props: { initialValue: boolean; disabled?: boolean }) => {
  const [value, setValue] = useState(props.initialValue);

  return (
    <BooleanCheckboxFormField
      name="enabled"
      label="Enabled"
      value={value}
      error={null}
      text="Enable notifications"
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const Unchecked: Story = {
  args: {} as BooleanCheckboxFormFieldProps,
  render: () => <StatefulCheckbox initialValue={false} />,
};

export const Checked: Story = {
  args: {} as BooleanCheckboxFormFieldProps,
  render: () => <StatefulCheckbox initialValue />,
};

export const Disabled: Story = {
  args: {} as BooleanCheckboxFormFieldProps,
  render: () => <StatefulCheckbox initialValue disabled />,
};
