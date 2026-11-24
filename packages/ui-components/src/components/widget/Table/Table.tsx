"use client";

import { useRef, useState } from "react";
import { TableProps } from "./Table.types";
import useTable from "./useTable";
import cn from "classnames";
import { TablePaginator } from "./components";
import { Spinner } from "@/components/common";
import { Form, FormLogic, FormValue, selectField, textField } from "@/components/form";

const Table = (props: TableProps) => {
  const tableLogic = useTable({ ...props });

  // States
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // Hooks
  const tableFormLogicRef = useRef<FormLogic | null>(null);

  return (
    <div className="table">
      {props.filters && props.filters.length > 0 && (
        <div className="table-filters">
          <Form
            fields={props.filters.map((filter) => {
              const onChange = (logic: FormLogic) => {
                const value = logic.fieldsState[filter.field].value;
                tableLogic.setFilterValue(filter.field, String(value ?? ""));
              };

              if (filter.type === "select") {
                return selectField({
                  name: filter.field,
                  label: filter.label,
                  value: tableLogic.filterValues[filter.field] ?? "",
                  placeholder: "All",
                  options: (filter.options ?? []).map((option) => ({
                    value: option.value,
                    displayed: <>{option.label}</>,
                  })),
                  error: null,
                  onChange,
                });
              }

              return textField({
                name: filter.field,
                label: filter.label,
                value: tableLogic.filterValues[filter.field] ?? "",
                placeholder: filter.placeholder,
                error: null,
                onChange,
              });
            })}
            onLogicReady={(logic) => {
              tableFormLogicRef.current = logic;
            }}
            onSubmit={(_values: Record<string, FormValue>) => undefined}
            submitButton={{ value: "Apply", className: "table-filter-submit" }}
          />
          {Object.values(tableLogic.filterValues).some(Boolean) && (
            <button
              type="button"
              className="table-filter-clear"
              onClick={() => {
                tableLogic.clearFilters();
                props.filters?.forEach((filter) =>
                  tableFormLogicRef.current?.setValue(filter.field, ""),
                );
              }}
            >
              Clear filters
            </button>
          )}
        </div>
      )}
      <div
        className={cn(
          "loading-wrapper",
          (isLoading || props.isLoading) && "visible",
        )}
      >
        {(isLoading || props.isLoading) && <Spinner />}
      </div>
      <table>
        <thead className="table-header">
          <tr>
            {props.rowLabels.map((label, i) => {
              return (
                <th
                  key={i}
                  className={cn("table-label")}
                  onClick={
                    props.orderBy == label.value
                      ? () => {
                          tableLogic.toggleSort();
                        }
                      : undefined
                  }
                  style={{
                    justifyContent:
                      props.rowLabels.length > 1 &&
                      i == props.rowLabels.length - 1
                        ? "flex-end"
                        : "flex-start",
                    cursor:
                      props.orderBy == label.value ? "pointer" : "default",
                  }}
                >
                  {label.label ?? label.value}
                  {props.orderBy == label.value && (
                    <span
                      className="material-symbols-outlined"
                      style={{
                        transform: tableLogic.sortedUp
                          ? "unset"
                          : "rotateX(180deg)",
                      }}
                    >
                      arrow_upward
                    </span>
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody
          className="table-rows"
          style={{
            height:
              props.isLoading || props.records.length <= 0
                ? `${75 * 5}px`
                : props.pageSize
                  ? `${75 * props.pageSize}px`
                  : "unset",
          }}
        >
          {tableLogic.records.length > 0 ? (
            tableLogic.records.map((row, i) => {
              return (
                <tr
                  key={row.id ?? i}
                  className={cn("table-row", props.onRowClick && "clickable")}
                  onClick={
                    props.onRowClick
                      ? async () => {
                          setIsLoading(true);
                          try {
                            await props.onRowClick?.(row);
                          } finally {
                            setIsLoading(false);
                          }
                        }
                      : undefined
                  }
                >
                  {props.rowLabels.map((label, j) => {
                    const item = row.items.find(
                      (val) => val.label == label.value,
                    );

                    return (
                      <td
                        key={j}
                        className="table-row-item"
                        style={{
                          justifyContent:
                            props.rowLabels.length > 1 &&
                            j == props.rowLabels.length - 1
                              ? "flex-end"
                              : "flex-start",
                        }}
                      >
                        {item && (item.content || item.value)}
                      </td>
                    );
                  })}
                </tr>
              );
            })
          ) : (
            <tr>
              {!props.isLoading && (
                <td className="material-symbols-outlined empty">search_off</td>
              )}
            </tr>
          )}
        </tbody>
      </table>
      {props.pageSize && tableLogic.pagesAmount > 1 ? (
        <TablePaginator tableLogic={tableLogic} />
      ) : (
        <div className="table-footer" />
      )}
    </div>
  );
};

export default Table;
