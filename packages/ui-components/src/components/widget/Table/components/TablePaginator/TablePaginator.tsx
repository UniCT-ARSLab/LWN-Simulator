import { Button, ButtonLayout, ButtonType } from "@/components/common";
import { TablePaginatorProps } from "./TablePaginator.types";
import cn from "classnames";

const TablePaginator = (props: TablePaginatorProps) => {
  return (
    <div className="table-paginator">
      <span>
        Page {props.tableLogic.currentPage} of {props.tableLogic.pagesAmount}
      </span>
      <div className="table-pages">
        <Button
          value={
            <span className="material-symbols-outlined">chevron_left</span>
          }
          type={ButtonType.Outlined}
          layout={ButtonLayout.Icon}
          onClick={() => {
            props.tableLogic.prevPage();
          }}
          disabled={props.tableLogic.currentPage == 1}
        />
        {props.tableLogic.visiblePages.map((page, i) => {
          return page >= 1 && page <= props.tableLogic.pagesAmount ? (
            <div
              key={i}
              className={cn(
                "table-page",
                page === props.tableLogic.currentPage && "current",
              )}
              onClick={
                page == props.tableLogic.currentPage
                  ? undefined
                  : () => {
                      props.tableLogic.setCurrentPage(page);
                    }
              }
            >
              {page}
            </div>
          ) : (
            <div key={i} className="page-placeholder" />
          );
        })}
        <Button
          value={
            <span className="material-symbols-outlined">chevron_right</span>
          }
          type={ButtonType.Outlined}
          layout={ButtonLayout.Icon}
          onClick={() => {
            props.tableLogic.nextPage();
          }}
          disabled={
            props.tableLogic.currentPage == props.tableLogic.pagesAmount
          }
        />
      </div>
    </div>
  );
};

export default TablePaginator;
