import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Button from "./Button";
import { ButtonFont, ButtonLayout, ButtonType } from "./Button.types";
import { NotificationHandler } from "@/utils";

const meta = {
  title: "ui-components/common/Button",
  component: Button,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleButton: Story = {
  args: {
    value: "Button Text",
    onClick: () => {
      console.log("Click");
    },
  },
};

export const SecondaryButton: Story = {
  args: {
    value: "Button Text",
    type: ButtonType.Secondary,
    onClick: (e) => {
      console.log("TARGET", e?.target);
      NotificationHandler.instance.success("Click");
    },
  },
};

export const OutlinedButton: Story = {
  args: {
    value: "Button Text",
    type: ButtonType.Outlined,
  },
};

export const BlockButton: Story = {
  args: {
    value: "Button Text",
    type: ButtonType.Outlined,
    layout: ButtonLayout.Block,
    font: ButtonFont.Secondary,
  },
};

export const LoadingButton: Story = {
  args: {
    value: "Button Text",
    type: ButtonType.Primary,
    font: ButtonFont.Secondary,
    onClick: async () => {
      await new Promise<void>((resolve) => {
        setTimeout(() => {
          resolve();
        }, 2000);
      });
    },
  },
};

export const IconButton: Story = {
  args: {
    value: <span className="material-symbols-outlined">chevron_right</span>,
    type: ButtonType.Outlined,
    font: ButtonFont.Secondary,
    layout: ButtonLayout.Icon,
    onClick: async () => {
      await new Promise<void>((resolve) => {
        setTimeout(() => {
          resolve();
        }, 2000);
      });
    },
  },
};

export const DisabledButton: Story = {
  args: {
    value: "Button Text",
    onClick: () => {
      console.log("Click");
    },
    disabled: true,
  },
};
