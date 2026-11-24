import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Popup from "./Popup";
import { Button } from "@/components/common";
import { usePopup } from "./usePopup";
import { PopupLogic } from "./Popup.types";

const meta = {
  title: "ui-components/widget/Popup",
  component: Popup,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Popup>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SimplePopup: Story = {
  args: {
    logic: {} as PopupLogic,
    title: "Title",
  },
  render: (args) => {
    const popupLogic = usePopup({});

    return (
      <>
        <Button
          value={"Open Popup"}
          onClick={() => {
            popupLogic.openPopup();
          }}
        />
        <Popup {...args} logic={popupLogic}>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
          <p>lorem ipsum dolor sit amet</p>
        </Popup>
      </>
    );
  },
};
