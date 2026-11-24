import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import EntityDetails from "./EntityDetails";
import {
  EntityDetailTone,
  EntityDetailValueFormat,
  EntityDetailsSectionLayout,
} from "./EntityDetails.types";

const meta = {
  title: "ui-components/layout/EntityDetails",
  component: EntityDetails,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof EntityDetails>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Overview: Story = {
  args: {
    sections: [
      {
        title: "Device overview",
        description: "Current runtime information.",
        icon: "sensors",
        layout: EntityDetailsSectionLayout.Half,
        metrics: [
          {
            label: "Packet success",
            value: "98.4%",
            detail: "Last 24 hours",
            tone: EntityDetailTone.Positive,
            icon: "check_circle",
          },
          {
            label: "Battery",
            value: "82%",
            detail: "3.58 V",
            tone: EntityDetailTone.Warning,
            icon: "battery_5_bar",
          },
        ],
        items: [
          { label: "Region", value: "EU868", mono: true },
          {
            label: "Activation",
            value: "over_the_air",
            format: EntityDetailValueFormat.Enum,
          },
        ],
      },
    ],
  },
};

export const MultipleSections: Story = {
  args: {
    sections: [
      {
        title: "Technical configuration",
        layout: EntityDetailsSectionLayout.Full,
        items: [
          { label: "DevEUI", value: "70B3D57ED0000001", mono: true },
          { label: "Class", value: "Class A" },
        ],
      },
      {
        title: "No recent activity",
        layout: EntityDetailsSectionLayout.Half,
        items: [],
        emptyState: {
          title: "Nothing to show",
          description: "This entity has not produced an event yet.",
          icon: "history",
        },
      },
    ],
  },
};
