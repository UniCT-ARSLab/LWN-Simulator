import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import SelectFormField from "./SelectFormField";
import { SelectFormFieldProps } from "./SelectFormField.types";

const options = [
  { value: "eu868", displayed: <>EU868</> },
  { value: "us915", displayed: <>US915</> },
  { value: "au915", displayed: <>AU915</> },
  { value: "as923", displayed: <>AS923</> },
  { value: "in865", displayed: <>IN865</> },
  { value: "kr920", displayed: <>KR920</> },
];

const largeOptions = Array.from({ length: 150 }, (_, index) => ({
  value: `option-${index + 1}`,
  displayed: <>Option {index + 1}</>,
}));

const meta = {
  title: "ui-components/form/fields/SelectFormField",
  component: SelectFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof SelectFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulSelect = (props: {
  initialValue: string | number | null;
  options: typeof options;
  disabled?: boolean;
}) => {
  const [value, setValue] = useState<string | number | null>(
    props.initialValue,
  );

  return (
    <SelectFormField
      name="region"
      label="Region"
      value={value}
      error={null}
      placeholder="Select a region"
      options={props.options}
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const Placeholder: Story = {
  args: {} as SelectFormFieldProps,
  render: () => <StatefulSelect initialValue={null} options={options} />,
};

export const SelectedValue: Story = {
  args: {} as SelectFormFieldProps,
  render: () => <StatefulSelect initialValue="eu868" options={options} />,
};

export const ManyOptions: Story = {
  args: {} as SelectFormFieldProps,
  render: () => (
    <StatefulSelect
      initialValue="option-75"
      options={largeOptions as typeof options}
    />
  ),
};

export const Disabled: Story = {
  args: {} as SelectFormFieldProps,
  render: () => (
    <StatefulSelect initialValue="us915" options={options} disabled />
  ),
};
