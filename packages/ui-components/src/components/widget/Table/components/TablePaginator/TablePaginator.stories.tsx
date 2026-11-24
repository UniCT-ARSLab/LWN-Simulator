import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import TablePaginator from "./TablePaginator";
import { TableLogic } from "../../Table.types";
import { TablePaginatorProps } from "./TablePaginator.types";

const meta = {
  title: "ui-components/widget/Table/TablePaginator",
  component: TablePaginator,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof TablePaginator>;

export default meta;
type Story = StoryObj<typeof meta>;

const createTableLogic = (
  currentPage: number,
  pagesAmount: number,
  setCurrentPage: (page: number) => void,
): TableLogic => ({
  records: [],
  toggleSort: () => undefined,
  sortedUp: true,
  pagesAmount,
  currentPage,
  nextPage: () => setCurrentPage(Math.min(currentPage + 1, pagesAmount)),
  prevPage: () => setCurrentPage(Math.max(currentPage - 1, 1)),
  setCurrentPage,
  filterValues: {},
  setFilterValue: () => undefined,
  clearFilters: () => undefined,
  visiblePages: Array.from({ length: 5 }, (_, index) => currentPage - 2 + index),
});

const InteractivePaginator = (props: { initialPage: number }) => {
  const [currentPage, setCurrentPage] = useState(props.initialPage);

  return (
    <TablePaginator
      tableLogic={createTableLogic(currentPage, 8, setCurrentPage)}
    />
  );
};

export const FirstPage: Story = {
  args: {} as TablePaginatorProps,
  render: () => <InteractivePaginator initialPage={1} />,
};

export const MiddlePage: Story = {
  args: {} as TablePaginatorProps,
  render: () => <InteractivePaginator initialPage={4} />,
};

export const LastPage: Story = {
  args: {} as TablePaginatorProps,
  render: () => <InteractivePaginator initialPage={8} />,
};
