"use client";

import { useEffect, useMemo, useState } from "react";
import { TableLogic, TableRecordItemSort, UseTableProps } from "./Table.types";

const useTable = (props: UseTableProps): TableLogic => {
  // States
  const [sortedUp, setSortedUp] = useState<boolean>(true);
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [filterValues, setFilterValues] = useState<Record<string, string>>({});

  // Memos
  const filteredRecords = useMemo(() => {
    const labels = new Set(props.rowLabels.map((label) => label.value));
    const activeFilters = (props.filters ?? []).filter(
      (filter) => filterValues[filter.field],
    );

    return props.records
      .map((record) => ({
        ...record,
        items: record.items.filter((item) => labels.has(item.label)),
      }))
      .filter((record) =>
        activeFilters.every((filter) => {
          const item = record.items.find((value) => value.label === filter.field);
          const value = item?.value.toLocaleLowerCase() ?? "";
          const filterValue = filterValues[filter.field].toLocaleLowerCase();

          return filter.type === "select"
            ? value === filterValue
            : value.includes(filterValue);
        }),
      );
  }, [filterValues, props.filters, props.records, props.rowLabels]);

  const sortedRecords = useMemo(() => {
    if (!props.orderBy) {
      return filteredRecords;
    }

    const label = props.rowLabels.find(
      (label) => label.value === props.orderBy,
    );

    if (!label) {
      return filteredRecords;
    }

    return [...filteredRecords].sort((a, b) => {
      const aValue = a.items.find(
        (item) => item.label === props.orderBy,
      )?.value;

      const bValue = b.items.find(
        (item) => item.label === props.orderBy,
      )?.value;

      if (aValue === undefined) return 1;
      if (bValue === undefined) return -1;

      let result = 0;

      switch (label.sort) {
        case TableRecordItemSort.Alphabetic:
          result = aValue.localeCompare(bValue, "en");
          break;

        case TableRecordItemSort.Numeric:
          result = Number(aValue) - Number(bValue);
          break;
      }

      return sortedUp ? result : -result;
    });
  }, [filteredRecords, props.orderBy, props.rowLabels, sortedUp]);

  const pagesAmount = useMemo(() => {
    if (!props.pageSize) return 1;

    return Math.max(1, Math.ceil(sortedRecords.length / props.pageSize));
  }, [props.pageSize, sortedRecords.length]);

  const visiblePages = useMemo(() => {
    return Array.from({ length: 5 }, (_, index) => {
      return currentPage - 2 + index;
    });
  }, [currentPage]);

  // Effects
  useEffect(() => {
    setCurrentPage((page) => Math.min(page, pagesAmount));
  }, [pagesAmount]);

  // Callbacks
  const setFilterValue = (field: string, value: string) => {
    setFilterValues((previous) => ({ ...previous, [field]: value }));
    setCurrentPage(1);
  };

  const clearFilters = () => {
    setFilterValues({});
    setCurrentPage(1);
  };

  const toggleSort = () => {
    setSortedUp((prev) => !prev);
  };

  const nextPage = () => {
    if (currentPage >= pagesAmount) return;
    setCurrentPage((prev) => prev + 1);
  };

  const prevPage = () => {
    if (currentPage <= 1) return;
    setCurrentPage((prev) => prev - 1);
  };

  const changePage = (page: number) => {
    setCurrentPage(Math.min(Math.max(page, 1), pagesAmount));
  };

  return {
    records: props.pageSize
      ? [...sortedRecords].slice(
          (currentPage - 1) * props.pageSize,
          (currentPage - 1) * props.pageSize + props.pageSize,
        )
      : sortedRecords,
    toggleSort,
    sortedUp,
    pagesAmount,
    currentPage,
    nextPage,
    prevPage,
    setCurrentPage: changePage,
    filterValues,
    setFilterValue,
    clearFilters,
    visiblePages,
  };
};

export default useTable;
