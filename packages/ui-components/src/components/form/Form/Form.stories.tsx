import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Form from "./Form";
import { Card, CardLayout } from "@/components/layout";
import {
  checkboxField,
  radioField,
  selectField,
  textAreaField,
  textField,
  booleanCheckboxField,
} from "./components";
import { FormField } from "./Form.types";
import { Button, ButtonLayout, ButtonType } from "@/components/common";

const meta = {
  title: "ui-components/form/Form",
  component: Form,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Form>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleForm: Story = {
  args: {
    fields: [
      textField({
        name: "name",
        label: "Name",
        value: "Name",
        info: { default: "Your Name" },
        error: null,
        toolbar: (
          <Button
            value={<span className="material-symbols-outlined">delete</span>}
            type={ButtonType.Outlined}
            layout={ButtonLayout.Icon}
            onClick={() => {
              console.log("Click");
            }}
          />
        ),
      }),
      textField({
        name: "last",
        label: "Last Name",
        value: "",
        placeholder: "Your last name",
        error: null,
      }),
      textField({
        name: "pass",
        label: "Password",
        value: "",
        placeholder: "Your Password",
        error: null,
        masked: true,
      }),
      textAreaField({
        name: "desc",
        label: "Description",
        value: "",
        placeholder: "About you",
        error: null,
        resize: true,
        charsMax: 20,
      }),
      selectField({
        name: "hobby",
        label: "Favorite Hobby",
        value: "",
        placeholder: "Your favorite hobby...",
        error: null,
        options: [
          {
            value: "drawing",
            displayed: <>Drawing</>,
          },
          {
            value: "running",
            displayed: (
              <>
                Running
                <span className="material-symbols-outlined">sprint</span>
              </>
            ),
          },
          {
            value: "fishing",
          },
        ],
      }),
      radioField({
        name: "color",
        label: "Favorite Color",
        value: "",
        error: null,
        options: [
          {
            value: "red",
            displayed: <>Red</>,
          },
          {
            value: "green",
            displayed: (
              <>
                Green
                <span
                  className="material-symbols-outlined"
                  style={{ color: "green", fontVariationSettings: "'FILL' 1" }}
                >
                  square
                </span>
              </>
            ),
          },
          {
            value: "blue",
          },
        ],
        info: {
          default: "color",
          red: "red",
          green: "Green",
        },
      }),
      checkboxField({
        name: "skills",
        label: "Your Skills",
        value: [],
        error: null,
        options: [
          {
            value: "html",
            displayed: <>HTML</>,
          },
          {
            value: "css",
            displayed: (
              <>
                CSS
                <span className="material-symbols-outlined">css</span>
              </>
            ),
          },
          {
            value: "js",
          },
        ],
      }),
      booleanCheckboxField({
        name: "conds",
        label: "Accept conditions",
        value: false,
        error: null,
        text: "Accept",
      }),
    ],
    onSubmit: (values) => {
      console.table(values);
    },
  },
  render: (args) => (
    <Card layout={CardLayout.Padded}>
      <Form {...args} />
    </Card>
  ),
};

export const ValidatedForm: Story = {
  args: {
    fields: [
      textField({
        name: "name",
        label: "Name",
        value: "Name",
        info: { default: "Your Name" },
        error: null,
        required: true,
      }),
      textField({
        name: "last",
        label: "Last Name",
        value: "",
        placeholder: "Your last name",
        error: null,
        format: (raw) => raw.toUpperCase(),
        validations: [
          { rule: /^.{8,}$/, error: "At least 8 chars" },
          { rule: /\d/, error: "At least 1 digit", realtime: true },
        ],
      }),
      textField({
        name: "pass",
        label: "Password",
        value: "",
        placeholder: "Your Password",
        error: null,
        masked: true,
      }),
      textAreaField({
        name: "desc",
        label: "Description",
        value: "",
        placeholder: "About you",
        error: null,
        resize: true,
        charsMax: 20,
        required: true,
        format: (raw) => raw.replaceAll(/[0-9]/g, ""),
      }),
      selectField({
        name: "hobby",
        label: "Favorite Hobby",
        value: "",
        placeholder: "Your favorite hobby...",
        error: null,
        required: true,

        options: [
          {
            value: "drawing",
            displayed: <>Drawing</>,
          },
          {
            value: "running",
            displayed: (
              <>
                Running
                <span className="material-symbols-outlined">sprint</span>
              </>
            ),
          },
          {
            value: "fishing",
          },
        ],
      }),
      radioField({
        name: "color",
        label: "Favorite Color",
        value: "",
        error: null,
        required: true,

        options: [
          {
            value: "red",
            displayed: <>Red</>,
          },
          {
            value: "green",
            displayed: (
              <>
                Green
                <span
                  className="material-symbols-outlined"
                  style={{ color: "green", fontVariationSettings: "'FILL' 1" }}
                >
                  square
                </span>
              </>
            ),
          },
          {
            value: "blue",
          },
        ],
      }),
      checkboxField({
        name: "skills",
        label: "Your Skills",
        value: [],
        error: null,
        required: true,

        options: [
          {
            value: "html",
            displayed: <>HTML</>,
          },
          {
            value: "css",
            displayed: (
              <>
                CSS
                <span className="material-symbols-outlined">css</span>
              </>
            ),
          },
          {
            value: "js",
          },
        ],
      }),
      booleanCheckboxField({
        name: "conds",
        label: "Accept conditions",
        value: false,
        error: null,
        text: "Accept",
        required: true,
      }),
    ],
    onSubmit: (values) => {
      console.table(values);
    },
  },
  render: (args) => (
    <Card layout={CardLayout.Padded}>
      <Form {...args} />
    </Card>
  ),
};

export const DisabledForm: Story = {
  args: {
    fields: [
      textField({
        name: "name",
        label: "Name",
        value: "Name",
        info: { default: "Your Name" },
        error: null,
        disabled: true,
      }),
      textField({
        name: "last",
        label: "Last Name",
        value: "",
        placeholder: "Your last name",
        error: null,
        disabled: true,
      }),
      textField({
        name: "pass",
        label: "Password",
        value: "",
        placeholder: "Your Password",
        error: null,
        masked: true,
        disabled: true,
      }),
      textAreaField({
        name: "desc",
        label: "Description",
        value: "",
        placeholder: "About you",
        error: null,
        resize: true,
        charsMax: 20,
        disabled: true,
      }),
      selectField({
        name: "hobby",
        label: "Favorite Hobby",
        value: "",
        placeholder: "Your favorite hobby...",
        error: null,
        options: [
          {
            value: "drawing",
            displayed: <>Drawing</>,
          },
          {
            value: "running",
            displayed: (
              <>
                Running
                <span className="material-symbols-outlined">sprint</span>
              </>
            ),
          },
          {
            value: "fishing",
          },
        ],
        disabled: true,
      }),
      radioField({
        name: "color",
        label: "Favorite Color",
        value: "",
        error: null,
        options: [
          {
            value: "red",
            displayed: <>Red</>,
          },
          {
            value: "green",
            displayed: (
              <>
                Green
                <span
                  className="material-symbols-outlined"
                  style={{ color: "green", fontVariationSettings: "'FILL' 1" }}
                >
                  square
                </span>
              </>
            ),
          },
          {
            value: "blue",
          },
        ],
        disabled: true,
      }),
      checkboxField({
        name: "skills",
        label: "Your Skills",
        value: [],
        error: null,
        options: [
          {
            value: "html",
            displayed: <>HTML</>,
          },
          {
            value: "css",
            displayed: (
              <>
                CSS
                <span className="material-symbols-outlined">css</span>
              </>
            ),
          },
          {
            value: "js",
          },
        ],
        disabled: true,
      }),
      booleanCheckboxField({
        name: "conds",
        label: "Accept conditions",
        value: false,
        error: null,
        text: "Accept",
        disabled: true,
      }),
    ],
    onSubmit: (values) => {
      console.table(values);
    },
  },
  render: (args) => (
    <Card layout={CardLayout.Padded}>
      <Form {...args} />
    </Card>
  ),
};

export const ConditionalForm: Story = {
  args: {
    fields: [
      textField({
        name: "name",
        label: "Name",
        value: "Name",
        info: { default: "Your Name" },
        error: null,
      }),
      textField({
        name: "last",
        label: "Last Name",
        value: "",
        placeholder: "Your last name",
        error: null,
        display: (fieldsState: Record<string, FormField>) => {
          return !!fieldsState["name"].value;
        },
      }),
      textField({
        name: "pass",
        label: "Password",
        value: "",
        placeholder: "Your Password",
        error: null,
        masked: true,
      }),
      textAreaField({
        name: "desc",
        label: "Description",
        value: "",
        placeholder: "About you",
        error: null,
        resize: true,
        charsMax: 20,
      }),
      selectField({
        name: "hobby",
        label: "Favorite Hobby",
        value: "",
        placeholder: "Your favorite hobby...",
        error: null,
        options: [
          {
            value: "drawing",
            displayed: <>Drawing</>,
          },
          {
            value: "running",
            displayed: (
              <>
                Running
                <span className="material-symbols-outlined">sprint</span>
              </>
            ),
          },
          {
            value: "fishing",
          },
        ],
      }),
      radioField({
        name: "color",
        label: "Favorite Color",
        value: "",
        error: null,
        options: [
          {
            value: "red",
            displayed: <>Red</>,
          },
          {
            value: "green",
            displayed: (
              <>
                Green
                <span
                  className="material-symbols-outlined"
                  style={{ color: "green", fontVariationSettings: "'FILL' 1" }}
                >
                  square
                </span>
              </>
            ),
          },
          {
            value: "blue",
          },
        ],
        display: (fieldsState: Record<string, FormField>) => {
          return !!fieldsState["desc"].value;
        },
      }),
      checkboxField({
        name: "skills",
        label: "Your Skills",
        value: [],
        error: null,
        options: [
          {
            value: "html",
            displayed: <>HTML</>,
          },
          {
            value: "css",
            displayed: (
              <>
                CSS
                <span className="material-symbols-outlined">css</span>
              </>
            ),
          },
          {
            value: "js",
          },
        ],
        disabled: (fieldsState: Record<string, FormField>) => {
          return fieldsState["hobby"].value == "running";
        },
        display: (fieldsState: Record<string, FormField>) => {
          return fieldsState["color"].value == "red";
        },
      }),
    ],
    onSubmit: (values) => {
      console.table(values);
    },
  },
  render: (args) => (
    <Card layout={CardLayout.Padded}>
      <Form {...args} />
    </Card>
  ),
};
