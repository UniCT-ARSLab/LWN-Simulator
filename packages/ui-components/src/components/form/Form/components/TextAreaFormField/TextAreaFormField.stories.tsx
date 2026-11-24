import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import TextAreaFormField from "./TextAreaFormField";
import { TextAreaFormFieldProps } from "./TextAreaFormField.types";

const meta = {
  title: "ui-components/form/fields/TextAreaFormField",
  component: TextAreaFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof TextAreaFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulTextArea = (props: {
  initialValue: string;
  charsMax?: number;
  resize?: boolean;
  disabled?: boolean;
}) => {
  const [value, setValue] = useState(props.initialValue);

  return (
    <TextAreaFormField
      name="description"
      label="Description"
      value={value}
      error={null}
      placeholder="Write a description"
      charsMax={props.charsMax}
      resize={props.resize}
      setValue={setValue}
      disabled={props.disabled}
    />
  );
};

export const Empty: Story = {
  args: {} as TextAreaFormFieldProps,
  render: () => <StatefulTextArea initialValue="" />,
};

export const WithCharacterLimit: Story = {
  args: {} as TextAreaFormFieldProps,
  render: () => (
    <StatefulTextArea
      initialValue="A text area with a character counter."
      charsMax={100}
      resize
    />
  ),
};

export const Disabled: Story = {
  args: {} as TextAreaFormFieldProps,
  render: () => (
    <StatefulTextArea initialValue="This field cannot be edited." disabled />
  ),
};
