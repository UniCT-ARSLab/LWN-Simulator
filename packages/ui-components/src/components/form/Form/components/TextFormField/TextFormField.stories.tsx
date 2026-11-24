import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import TextFormField from "./TextFormField";
import { TextFormFieldProps } from "./TextFormField.types";

const meta = {
  title: "ui-components/form/fields/TextFormField",
  component: TextFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof TextFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulTextField = (props: {
  initialValue: string;
  masked?: boolean;
  readOnly?: boolean;
  disabled?: boolean;
}) => {
  const [value, setValue] = useState(props.initialValue);

  return (
    <TextFormField
      name="value"
      label="Value"
      value={value}
      error={null}
      placeholder="Enter a value"
      masked={props.masked}
      readOnly={props.readOnly}
      disabled={props.disabled}
      setValue={setValue}
    />
  );
};

export const PlainText: Story = {
  args: {} as TextFormFieldProps,
  render: () => <StatefulTextField initialValue="Editable text" />,
};

export const Masked: Story = {
  args: {} as TextFormFieldProps,
  render: () => (
    <StatefulTextField initialValue="secret-value" masked />
  ),
};

export const ReadOnly: Story = {
  args: {} as TextFormFieldProps,
  render: () => (
    <StatefulTextField initialValue="Read-only text" readOnly />
  ),
};

export const Disabled: Story = {
  args: {} as TextFormFieldProps,
  render: () => (
    <StatefulTextField initialValue="Disabled text" disabled />
  ),
};
