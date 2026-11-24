import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import Table from "./Table";
import { TableRecord, TableRecordItemSort } from "./Table.types";

const meta = {
  title: "ui-components/widget/Table",
  component: Table,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof Table>;

export default meta;
type Story = StoryObj<typeof meta>;

const testRecords = [
  {
    items: [
      { label: "age", value: "30" },
      { label: "id", value: "0", content: <strong>00</strong> },
      { label: "name", value: "John" },
      { label: "wrong", value: "Wrong" },
    ],
  },
  {
    items: [
      { label: "id", value: "1" },
      { label: "name", value: "Anna" },
    ],
  },
  {
    items: [
      { label: "id", value: "2" },
      { label: "name", value: "Marco" },
      { label: "age", value: "42" },
    ],
  },
  {
    items: [
      { label: "name", value: "Laura" },
      { label: "age", value: "28" },
    ],
  },
  {
    items: [
      { label: "id", value: "4" },
      { label: "name", value: "David" },
      { label: "age", value: "35" },
    ],
  },
  {
    items: [
      { label: "id", value: "5" },
      { label: "name", value: "Sofia" },
      { label: "age", value: "31" },
    ],
  },
  {
    items: [
      { label: "id", value: "6" },
      { label: "name", value: "Luca" },
      { label: "age", value: "22" },
    ],
  },
  {
    items: [
      { label: "id", value: "7" },
      { label: "name", value: "Emma" },
      { label: "age", value: "27" },
    ],
  },
  {
    items: [
      { label: "id", value: "8" },
      { label: "name", value: "Alex" },
      { label: "age", value: "39" },
    ],
  },
  {
    items: [
      { label: "id", value: "9" },
      { label: "name", value: "Giulia" },
      { label: "age", value: "24" },
    ],
  },
  {
    items: [
      { label: "name", value: "Tom" },
      { label: "age", value: "45" },
    ],
  },
  {
    items: [
      { label: "id", value: "11" },
      { label: "name", value: "Elena" },
      { label: "age", value: "33" },
    ],
  },
  {
    items: [
      { label: "id", value: "12" },
      { label: "name", value: "Paul" },
      { label: "age", value: "29" },
    ],
  },
  {
    items: [
      { label: "id", value: "13" },
      { label: "name", value: "Sara" },
      { label: "age", value: "36" },
    ],
  },
  {
    items: [
      { label: "id", value: "14" },
      { label: "name", value: "Mike" },
      { label: "age", value: "41" },
    ],
  },
  {
    items: [
      { label: "id", value: "15" },
      { label: "name", value: "Chiara" },
      { label: "age", value: "26" },
    ],
  },
  {
    items: [
      { label: "id", value: "16" },
      { label: "name", value: "Andrea" },
      { label: "age", value: "38" },
    ],
  },
  {
    items: [
      { label: "id", value: "17" },
      { label: "name", value: "Nina" },
      { label: "age", value: "23" },
    ],
  },
  {
    items: [
      { label: "id", value: "18" },
      { label: "name", value: "Robert" },
      { label: "age", value: "50" },
    ],
  },
  {
    items: [
      { label: "name", value: "Marta" },
      { label: "age", value: "32" },
    ],
  },
  {
    items: [
      { label: "id", value: "20" },
      { label: "name", value: "Kevin" },
      { label: "age", value: "37" },
    ],
  },
  {
    items: [
      { label: "id", value: "21" },
      { label: "name", value: "Alice" },
      { label: "age", value: "34" },
    ],
  },
  {
    items: [
      { label: "id", value: "22" },
      { label: "name", value: "Fabio" },
      { label: "age", value: "40" },
    ],
  },
  {
    items: [
      { label: "id", value: "23" },
      { label: "name", value: "Maya" },
      { label: "age", value: "21" },
    ],
  },
  {
    items: [
      { label: "id", value: "24" },
      { label: "name", value: "Chris" },
      { label: "age", value: "44" },
    ],
  },
  {
    items: [
      { label: "id", value: "25" },
      { label: "name", value: "Paola" },
      { label: "age", value: "30" },
    ],
  },
  {
    items: [
      { label: "name", value: "Leo" },
      { label: "age", value: "27" },
    ],
  },
  {
    items: [
      { label: "id", value: "27" },
      { label: "name", value: "Irene" },
      { label: "age", value: "35" },
    ],
  },
  {
    items: [
      { label: "id", value: "28" },
      { label: "name", value: "Daniel" },
      { label: "age", value: "43" },
    ],
  },
  {
    items: [
      { label: "id", value: "29" },
      { label: "name", value: "Valentina" },
      { label: "age", value: "28" },
    ],
  },
  {
    items: [
      { label: "id", value: "30" },
      { label: "name", value: "Simon" },
      { label: "age", value: "31" },
    ],
  },
  {
    items: [
      { label: "id", value: "31" },
      { label: "name", value: "Clara" },
      { label: "age", value: "26" },
    ],
  },
  {
    items: [
      { label: "id", value: "32" },
      { label: "name", value: "Matteo" },
      { label: "age", value: "39" },
    ],
  },
  {
    items: [
      { label: "id", value: "33" },
      { label: "name", value: "Olivia" },
      { label: "age", value: "29" },
    ],
  },
  {
    items: [
      { label: "id", value: "34" },
      { label: "name", value: "James" },
      { label: "age", value: "46" },
    ],
  },
  {
    items: [
      { label: "id", value: "35" },
      { label: "name", value: "Beatrice" },
      { label: "age", value: "25" },
    ],
  },
  {
    items: [
      { label: "id", value: "36" },
      { label: "age", value: "48" },
    ],
  },
  {
    items: [
      { label: "id", value: "37" },
      { label: "age", value: "37" },
    ],
  },
  {
    items: [
      { label: "id", value: "38" },
      { label: "age", value: "33" },
    ],
  },
  {
    items: [
      { label: "id", value: "39" },
      { label: "name", value: "Francesca" },
      { label: "age", value: "42" },
    ],
  },
  {
    items: [
      { label: "id", value: "40" },
      { label: "name", value: "George" },
      { label: "age", value: "36" },
    ],
  },
  {
    items: [
      { label: "id", value: "41" },
      { label: "name", value: "Elisa" },
      { label: "age", value: "24" },
    ],
  },
  {
    items: [
      { label: "id", value: "42" },
      { label: "name", value: "Victor" },
      { label: "age", value: "52" },
    ],
  },
  {
    items: [
      { label: "id", value: "43" },
      { label: "name", value: "Noemi" },
      { label: "age", value: "30" },
    ],
  },
  {
    items: [
      { label: "id", value: "44" },
      { label: "name", value: "Henry" },
      { label: "age", value: "47" },
    ],
  },
  {
    items: [
      { label: "id", value: "45" },
      { label: "name", value: "Alice" },
      { label: "age", value: "28" },
    ],
  },
  {
    items: [
      { label: "id", value: "46" },
      { label: "name", value: "Stefano" },
      { label: "age", value: "34" },
    ],
  },
  {
    items: [
      { label: "id", value: "47" },
      { label: "name", value: "Laura" },
      { label: "age", value: "40" },
    ],
  },
  {
    items: [
      { label: "name", value: "Thomas" },
      { label: "age", value: "38" },
    ],
  },
  {
    items: [
      { label: "id", value: "49" },
      { label: "name", value: "Elena" },
      { label: "age", value: "27" },
    ],
  },
];

export const SimpleTable: Story = {
  args: {
    rowLabels: [{ value: "id" }, { value: "name" }, { value: "age" }],
    records: testRecords.slice(0, 5),
  },
};

export const EmptyTable: Story = {
  args: {
    rowLabels: [{ value: "id" }, { value: "name" }, { value: "age" }],
    records: [],
  },
};

export const ClickableTable: Story = {
  args: {
    rowLabels: [{ value: "id" }, { value: "name" }, { value: "age" }],
    records: testRecords.slice(0, 5),
    onRowClick: async (row: TableRecord) => {
      await new Promise<void>((resolve) => {
        setTimeout(() => {
          console.table(row);
          resolve();
        }, 2000);
      });
    },
  },
};

export const LoadingTable: Story = {
  args: {
    rowLabels: [{ value: "id" }, { value: "name" }, { value: "age" }],
    records: [],
    isLoading: true,
  },
};

export const ScrollableTable: Story = {
  args: {
    rowLabels: [{ value: "id" }, { value: "name" }, { value: "age" }],
    records: testRecords,
  },
  render: (args) => (
    <div style={{ height: "calc(100vh - 4rem)" }}>
      <Table {...args} />
    </div>
  ),
};

export const NumericSortedTable: Story = {
  args: {
    rowLabels: [
      { value: "id", sort: TableRecordItemSort.Numeric },
      { value: "name" },
      { value: "age" },
    ],
    records: testRecords,
    orderBy: "id",
  },
  render: (args) => (
    <div style={{ height: "calc(100vh - 4rem)" }}>
      <Table {...args} />
    </div>
  ),
};

export const AlphabeticSortedTable: Story = {
  args: {
    rowLabels: [
      { value: "id" },
      { value: "name", sort: TableRecordItemSort.Alphabetic },
      { value: "age" },
    ],
    records: testRecords,
    orderBy: "name",
  },
  render: (args) => (
    <div style={{ height: "calc(100vh - 4rem)" }}>
      <Table {...args} />
    </div>
  ),
};

export const PaginatedTable: Story = {
  args: {
    rowLabels: [
      { value: "id" },
      { value: "name", sort: TableRecordItemSort.Alphabetic },
      { value: "age" },
    ],
    records: testRecords,
    pageSize: 7,
  },
};
