import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import ProgressBar from "./ProgressBar";
import { ProgressBarType } from "./ProgressBar.types";

const meta = {
  title: "ui-components/common/ProgressBar",
  component: ProgressBar,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof ProgressBar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleProgressBar: Story = {
  args: {
    percentage: 50,
  },
};

export const SecondaryProgressBar: Story = {
  args: {
    percentage: 86.45,
    type: ProgressBarType.Secondary,
  },
};
