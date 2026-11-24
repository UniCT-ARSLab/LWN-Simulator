import { JSX } from "react/jsx-runtime";

/**
 * Defines the supported sorting strategies for table columns.
 */
export enum TableRecordItemSort {
  /** Sorts values using numeric comparison. */
  Numeric = 1,

  /** Sorts values using alphabetical comparison. */
  Alphabetic,
}

/**
 * Represents the configuration of a table column label.
 */
export interface TableLabel {
  /**
   * Text displayed as the column header.
   */
  value: string;

  /** Optional human-readable header label. */
  label?: string;

  /**
   * Sorting strategy applied to the column.
   *
   * When omitted, the column does not support sorting.
   */
  sort?: TableRecordItemSort;
}

/**
 * Represents a single cell value inside a table row.
 */
export interface TableRecordItem {
  /**
   * Identifier of the cell, matching the related column label.
   */
  label: string;

  /**
   * Raw value associated with the cell.
   */
  value: string;

  /**
   * Custom React element rendered instead of the raw value.
   */
  content?: JSX.Element;
}

/**
 * Represents a single row of data displayed in the table.
 */
export interface TableRecord {
  /** Stable identifier used as the React key and by row actions. */
  id?: string;

  /**
   * Collection of cells composing the row.
   */
  items: Array<TableRecordItem>;
}

/** Input control types supported by table filters. */
export type TableFilterType = "text" | "select";

/** Option displayed by a select-based table filter. */
export interface TableFilterOption {
  /** Text shown to the user. */
  label: string;

  /** Value applied to the filter when selected. */
  value: string;
}

/** Configuration for one table filter control. */
export interface TableFilter {
  /** Row item field used for filtering. */
  field: string;

  /** Label displayed above the filter control. */
  label: string;

  /** Filter control type. */
  type?: TableFilterType;

  /** Placeholder shown by a text filter. */
  placeholder?: string;

  /** Options shown by a select filter. */
  options?: Array<TableFilterOption>;
}

/** Properties for configuring a table component. */
export interface TableProps {
  /**
   * Column definitions displayed in the table header.
   */
  rowLabels: Array<TableLabel>;

  /**
   * Data rows rendered inside the table.
   */
  records: Array<TableRecord>;

  /**
   * Column label used as the default sorting key.
   */
  orderBy?: string;

  /**
   * Callback executed when a table row is clicked.
   *
   * Supports both synchronous and asynchronous handlers.
   */
  onRowClick?: (row: TableRecord) => void | Promise<void>;

  /** Filters rendered above the table. */
  filters?: Array<TableFilter>;

  /**
   * Number of rows displayed per page.
   *
   * When provided, pagination is enabled and the table displays the configured
   * number of records per page.
   *
   * When omitted, pagination is disabled and all records are displayed on a
   * single page without showing pagination controls.
   */
  pageSize?: number;

  /**
   * Indicates whether the table is currently loading.
   *
   * When enabled, the table displays its loading state instead of the
   * available records.
   *
   * @default false
   */
  isLoading?: boolean;
}

/**
 * Properties accepted by the `useTable` hook.
 *
 * Extends the table configuration properties.
 */
export interface UseTableProps extends TableProps {}

/**
 * State and actions exposed by the table logic hook.
 */
export interface TableLogic {
  /**
   * Current table records after applying sorting and pagination logic.
   */
  records: Array<TableRecord>;

  /**
   * Toggles the current sorting direction.
   */
  toggleSort: () => void;

  /**
   * Indicates whether records are currently sorted in ascending order.
   */
  sortedUp: boolean;

  /**
   * Total number of available pages.
   */
  pagesAmount: number;

  /**
   * Currently selected page index.
   */
  currentPage: number;

  /**
   * Navigates to the next available page.
   */
  nextPage: () => void;

  /**
   * Navigates to the previous available page.
   */
  prevPage: () => void;

  /**
   * Changes the currently selected page.
   *
   * @param page The target page index.
   */
  setCurrentPage: (page: number) => void;

  /** Current values of the configured filters. */
  filterValues: Record<string, string>;

  /** Updates one filter value. */
  setFilterValue: (field: string, value: string) => void;

  /** Clears all active filters. */
  clearFilters: () => void;

  /**
   * List of page indexes currently visible in the pagination controls.
   */
  visiblePages: Array<number>;
}
