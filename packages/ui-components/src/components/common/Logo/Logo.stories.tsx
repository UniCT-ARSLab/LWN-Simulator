import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Logo from "./Logo";
import { LogoLayout, LogoSize, LogoType } from "./Logo.types";

const meta = {
  title: "ui-components/common/Logo",
  component: Logo,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Logo>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleLogo: Story = {
  args: {
    size: LogoSize.Lg,
  },
};

export const SecondaryLogo: Story = {
  args: {
    type: LogoType.Secondary,
    layout: LogoLayout.SecondaryBackground,
    size: LogoSize.Lg,
  },
};
