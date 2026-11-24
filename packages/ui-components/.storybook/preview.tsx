/* eslint-disable @typescript-eslint/ban-ts-comment */
import type { Preview } from "@storybook/nextjs-vite";
// @ts-expect-error
import "./storybook.scss";
// @ts-expect-error
import "../src/styles/main.scss";
import { ArgTypesEnhancer } from "storybook/internal/types";
import { ToastContainer } from "react-toastify";

const enumEnhancer: ArgTypesEnhancer = ({ argTypes }) => {
  return Object.fromEntries(
    Object.entries(argTypes).map(([name, argType]) => {
      const type = argType.type;

      if (
        type?.name === "enum" &&
        Array.isArray(type.value) &&
        type.value.length > 0
      ) {
        const originalValues = type.value.map(String);

        return [
          name,
          {
            ...argType,
            control: {
              type: "radio",
            },
            options: originalValues.map((v) =>
              v
                .replaceAll('"', "")
                .replace(/^.*\./, "")
                .replaceAll("-", " ")
                .replaceAll(/(var\(|\))/g, "")
                .replace(/^\s*([a-z])/, (c) => c.toUpperCase()),
            ),
            mapping: Object.fromEntries(
              originalValues.map((value, index) => [
                originalValues[index]
                  .replaceAll('"', "")
                  .replace(/^.*\./, "")
                  .replaceAll("-", " ")
                  .replaceAll(/(var\(|\))/g, "")
                  .replace(/^\s*([a-z])/, (c) => c.toUpperCase()),
                value.replaceAll('"', ""),
              ]),
            ),
          },
        ];
      }

      return [name, argType];
    }),
  );
};

const preview: Preview = {
  argTypesEnhancers: [enumEnhancer],

  globalTypes: {
    theme: {
      name: "Theme",
      description: "Global Theme",
      defaultValue: "light-theme",
      toolbar: {
        icon: "mirror",
        items: ["light-theme", "dark-theme"],
      },
    },
  },

  decorators: [
    (Story, context) => {
      const theme = context.globals.theme;

      return (
        <div className={`sb-wrapper-custom ${theme}`}>
          <Story />
          <ToastContainer />
        </div>
      );
    },
  ],
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
};

export default preview;
