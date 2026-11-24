import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import PageHeader from "./PageHeader";
import { Button } from "@/components/common";

const meta = {
  title: "ui-components/layout/PageHeader",
  component: PageHeader,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof PageHeader>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimplePageHeader: Story = {
  args: {
    title: "Page title",
  },
};

export const SubTitlePageHeader: Story = {
  args: {
    title: "Page title",
    subTitle: "Page sub title",
  },
};

export const ChildrenPageHeader: Story = {
  args: {
    children: (
      <>
        <Button value={"Button 1"} />
        <Button value={"Button 2"} />
        <Button value={"Button 3"} />
      </>
    ),
    title: "Page title",
    subTitle: "Page sub title",
  },
};
