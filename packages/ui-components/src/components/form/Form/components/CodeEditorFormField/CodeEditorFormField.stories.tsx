import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import CodeEditorFormField from "./CodeEditorFormField";
import { CodeEditorFormFieldProps } from "./CodeEditorFormField.types";

const meta = {
  title: "ui-components/form/fields/CodeEditorFormField",
  component: CodeEditorFormField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof CodeEditorFormField>;

export default meta;
type Story = StoryObj<typeof meta>;

const StatefulCodeEditor = (props: {
  initialValue: string;
  charsMax?: number;
  resize?: boolean;
  disabled?: boolean;
  readOnly?: boolean;
}) => {
  const [value, setValue] = useState(props.initialValue);

  return (
    <CodeEditorFormField
      name="codec"
      label="Payload codec"
      value={value}
      error={null}
      placeholder="Paste JavaScript codec code"
      charsMax={props.charsMax}
      resize={props.resize}
      disabled={props.disabled}
      readOnly={props.readOnly}
      setValue={setValue}
    />
  );
};

export const Empty: Story = {
  args: {} as CodeEditorFormFieldProps,
  render: () => <StatefulCodeEditor initialValue="" resize />,
};

export const JavaScript: Story = {
  args: {} as CodeEditorFormFieldProps,
  render: () => (
    <StatefulCodeEditor
      initialValue={`function decodeUplink(input) {\n  return { data: {} };\n}`}
      resize
    />
  ),
};

export const WithCharacterLimit: Story = {
  args: {} as CodeEditorFormFieldProps,
  render: () => (
    <StatefulCodeEditor
      initialValue="function decodeUplink(input) {\n  return { data: {} };\n}"
      charsMax={200}
      resize
    />
  ),
};

export const Disabled: Story = {
  args: {} as CodeEditorFormFieldProps,
  render: () => (
    <StatefulCodeEditor
      initialValue="function decodeUplink(input) {\n  return { data: {} };\n}"
      disabled
    />
  ),
};

export const ReadOnly: Story = {
  args: {} as CodeEditorFormFieldProps,
  render: () => (
    <StatefulCodeEditor
      initialValue="This codec is visible but cannot be edited."
      readOnly
    />
  ),
};
