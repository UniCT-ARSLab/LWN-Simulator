import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import AnimatedField from "./AnimatedField";
import { FormField, FormLogic } from "../../Form.types";
import { AnimatedFieldProps } from "./AnimatedField.types";
import { Button } from "@/components/common";

const meta = {
  title: "ui-components/form/fields/AnimatedField",
  component: AnimatedField,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof AnimatedField>;

export default meta;
type Story = StoryObj<typeof meta>;

const AnimatedFieldStory = (props: { hasError?: boolean }) => {
  const [value, setValue] = useState("");
  const [visible, setVisible] = useState(true);
  const field: FormField = {
    name: "name",
    label: "Name",
    value,
    error: props.hasError ? "This value needs attention" : null,
    required: true,
    info: { default: "Enter a display name." },
    toolbar: <Button value="Clear" onClick={() => setValue("")} />,
    render: ({ value: fieldValue, setValue: updateValue }) => (
      <input
        className="form-field text-form-field"
        value={String(fieldValue ?? "")}
        onChange={(event) => updateValue(event.target.value)}
      />
    ),
  };
  const formLogic: FormLogic = {
    fieldsState: { name: field },
    setValue: (_name, nextValue) => setValue(String(nextValue ?? "")),
    setProp: () => undefined,
    isFieldDisabled: () => false,
    isFieldDisplayed: () => visible,
    validate: () => !props.hasError,
  };

  return (
    <>
      <Button
        value={visible ? "Hide field" : "Show field"}
        onClick={() => setVisible((current) => !current)}
      />
      <AnimatedField field={field} visible={visible} formLogic={formLogic} />
    </>
  );
};

export const VisibleField: Story = {
  args: {} as AnimatedFieldProps,
  render: () => <AnimatedFieldStory />,
};

export const FieldWithError: Story = {
  args: {} as AnimatedFieldProps,
  render: () => <AnimatedFieldStory hasError />,
};
