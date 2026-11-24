import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Card from "./Card";
import { CardLayout, CardType } from "./Card.types";

const meta = {
  title: "ui-components/layout/Card",
  component: Card,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Card>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimpleCard: Story = {
  args: {
    children: (
      <>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
      </>
    ),
  },
};

export const PaddedCard: Story = {
  args: {
    children: (
      <>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
      </>
    ),
    layout: CardLayout.Padded,
  },
};

export const TransparentCard: Story = {
  args: {
    children: (
      <>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
        <p>
          Lorem ipsum dolor sit amet consectetur adipisicing elit. Laboriosam
          tenetur totam cum labore saepe esse culpa architecto. Totam tempora
          explicabo praesentium tempore dignissimos aperiam expedita. Nostrum
          enim accusamus inventore earum!
        </p>
      </>
    ),
    layout: CardLayout.Padded,
    type: CardType.Transparent,
  },
};
