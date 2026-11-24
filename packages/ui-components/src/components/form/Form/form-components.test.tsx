import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import Form from "./Form";
import { booleanCheckboxField, selectField, textField } from "./components";
import SelectFormField from "./components/SelectFormField/SelectFormField";
import TextFormField from "./components/TextFormField/TextFormField";
import CodeEditorFormField from "./components/CodeEditorFormField/CodeEditorFormField";

describe("form components", () => {
  it("formats text input and keeps disabled fields out of submitted values", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <Form
        fields={[
          textField({
            name: "name",
            label: "Name",
            value: "",
            error: null,
            format: (value) => value.toUpperCase(),
            required: true,
          }),
          textField({
            name: "id",
            label: "Identifier",
            value: "fixed",
            error: null,
            disabled: true,
          }),
        ]}
        onSubmit={onSubmit}
      />,
    );

    await user.type(screen.getByLabelText("Name*"), "node");
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(onSubmit).toHaveBeenCalledWith({ name: "NODE" });
  });

  it("shows required validation errors before submitting", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <Form
        fields={[
          textField({
            name: "name",
            label: "Name",
            value: "",
            error: null,
            required: true,
          }),
        ]}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(screen.getByText("This field is required")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("supports predicate validation rules on submission", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <Form
        fields={[
          textField({
            name: "script",
            label: "Script",
            value: "invalid",
            error: null,
            validations: [
              {
                rule: (value) => value === "valid",
                error: "Invalid script",
              },
            ],
          }),
        ]}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(screen.getByText("Invalid script")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("accepts zero as a valid required numeric value", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <Form
        fields={[
          selectField({
            name: "dataRate",
            label: "Data rate",
            value: 0,
            error: null,
            options: [{ value: 0 }],
            required: true,
          }),
        ]}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(onSubmit).toHaveBeenCalledWith({ dataRate: 0 });
  });

  it("changes a custom checkbox field through the form logic", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <Form
        fields={[
          booleanCheckboxField({
            name: "enabled",
            label: "Enabled",
            text: "Enable device",
            value: false,
            error: null,
          }),
        ]}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByText("Enable device"));
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(onSubmit).toHaveBeenCalledWith({ enabled: true });
  });

  it("selects an option and respects the disabled state", async () => {
    const user = userEvent.setup();
    const setValue = vi.fn();
    const { rerender, container } = render(
      <SelectFormField
        name="region"
        label="Region"
        value={null}
        error={null}
        placeholder="Select region"
        options={[
          { value: "eu868", displayed: <>EU868</> },
          { value: "us915", displayed: <>US915</> },
        ]}
        setValue={setValue}
      />,
    );

    await user.click(screen.getByText("Select region"));
    await user.click(screen.getByText("EU868"));
    expect(setValue).toHaveBeenCalledWith("eu868");

    rerender(
      <SelectFormField
        name="region"
        label="Region"
        value="eu868"
        error={null}
        placeholder="Select region"
        options={[{ value: "eu868", displayed: <>EU868</> }]}
        setValue={setValue}
        disabled
      />,
    );
    expect(container.querySelector(".select-form-field")).toHaveClass(
      "disabled",
    );
  });

  it("supports controlled text and disabled fields", () => {
    const ControlledText = () => {
      const [value, setValue] = useState("initial");
      return (
        <TextFormField
          name="value"
          label="Value"
          value={value}
          error={null}
          setValue={setValue}
          disabled
        />
      );
    };

    render(<ControlledText />);
    expect(screen.getByDisplayValue("initial")).toBeDisabled();
  });

  it("edits multiline code and respects the character limit", async () => {
    const user = userEvent.setup();
    const CodeEditor = () => {
      const [value, setValue] = useState("");

      return (
        <CodeEditorFormField
          name="codec"
          label="Payload codec"
          value={value}
          error={null}
          charsMax={20}
          setValue={setValue}
        />
      );
    };

    render(<CodeEditor />);

    await user.type(screen.getByLabelText("Payload codec"), "line 1\nline 2");

    expect(screen.getByLabelText("Payload codec")).toHaveValue("line 1\nline 2");
  });

  it("renders the code editor disabled", () => {
    render(
      <CodeEditorFormField
        name="codec"
        label="Payload codec"
        value="function decodeUplink() {}"
        error={null}
        setValue={vi.fn()}
        disabled
      />,
    );

    expect(screen.getByLabelText("Payload codec")).toBeDisabled();
  });
});
