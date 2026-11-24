import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import RadioFormField from "./RadioFormField";
import { RadioFormFieldProps } from "./RadioFormField.types";

const options = [
  { value: "low", displayed: <>Low</> },
  { value: "medium", displayed: <>Medium</> },
  { value: "high", displayed: <>High</> },
];

const meta = {
  title: "ui-components/form/fields/RadioFormField",
  component: RadioFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof RadioFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulRadio = (props: { initialValue: string | null; disabled?: boolean }) => {
  const [value, setValue] = useState<string | null>(props.initialValue);

  return (
    <RadioFormField
      name="priority"
      label="Priority"
      value={value}
      error={null}
      options={options}
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const NoSelection: Story = {
  args: {} as RadioFormFieldProps,
  render: () => <StatefulRadio initialValue={null} />,
};

export const Selected: Story = {
  args: {} as RadioFormFieldProps,
  render: () => <StatefulRadio initialValue="medium" />,
};

export const Disabled: Story = {
  args: {} as RadioFormFieldProps,
  render: () => <StatefulRadio initialValue="high" disabled />,
};
