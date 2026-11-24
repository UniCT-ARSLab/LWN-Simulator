import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import CheckboxFormField from "./CheckboxFormField";
import { CheckboxFormFieldProps } from "./CheckboxFormField.types";

const options = [
  { value: "email", displayed: <>Email</> },
  { value: "sms", displayed: <>SMS</> },
  { value: "push", displayed: <>Push notification</> },
];

const meta = {
  title: "ui-components/form/fields/CheckboxFormField",
  component: CheckboxFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof CheckboxFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulCheckboxes = (props: { initialValue: Array<string> | null; disabled?: boolean }) => {
  const [value, setValue] = useState<Array<string> | null>(props.initialValue);

  return (
    <CheckboxFormField
      name="channels"
      label="Channels"
      value={value}
      error={null}
      options={options}
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const EmptySelection: Story = {
  args: {} as CheckboxFormFieldProps,
  render: () => <StatefulCheckboxes initialValue={null} />,
};

export const MultipleSelection: Story = {
  args: {} as CheckboxFormFieldProps,
  render: () => <StatefulCheckboxes initialValue={["email", "push"]} />,
};

export const Disabled: Story = {
  args: {} as CheckboxFormFieldProps,
  render: () => <StatefulCheckboxes initialValue={["email"]} disabled />,
};
