import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Spinner from "./Spinner";
import { SpinnerSize, SpinnerType } from "./Spinner.types";

const meta = {
  title: "ui-components/common/Spinner",
  component: Spinner,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Spinner>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleSpinner: Story = {
  args: {
    size: SpinnerSize.Md,
  },
};

export const SecondarySpinner: Story = {
  args: {
    size: SpinnerSize.Lg,
    type: SpinnerType.Secondary,
  },
};
