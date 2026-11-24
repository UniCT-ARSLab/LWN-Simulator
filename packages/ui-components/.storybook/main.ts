import type { StorybookConfig } from "@storybook/nextjs-vite";

import { dirname } from "path";
import { fileURLToPath } from "url";

function getAbsolutePath(value: string) {
  return dirname(fileURLToPath(import.meta.resolve(`${value}/package.json`)));
}

const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],

  addons: ["@storybook/addon-docs"],

  framework: getAbsolutePath("@storybook/nextjs-vite"),

  typescript: {
    reactDocgen: "react-docgen-typescript",
  },

  features: {
    argTypeTargetsV7: true,
  },

  staticDirs: ["../public"],

  viteFinal: async (config, { configType }) => {
    if (configType === "PRODUCTION") {
      config.base = "/LWN-Simulator/storybook/";
    } else {
      config.base = "/";
    }

    return config;
  },

  managerHead: (head, { configType }) => {
    if (configType === "PRODUCTION") {
      return `
        ${head}
        <base href="/LWN-Simulator/storybook/">
      `;
    }

    return head;
  },
};

export default config;
