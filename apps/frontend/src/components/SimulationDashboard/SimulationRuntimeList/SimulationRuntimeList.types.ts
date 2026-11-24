import { ReactNode } from "react";

/** Properties accepted by the virtualized runtime list. */
export interface SimulationRuntimeListProps<T extends { id: string }> {
  /** Runtime items displayed by the list. */
  items: Array<T>;
  /** Message shown when the list has no items. */
  emptyMessage: string;
  /** Renders the contents of one runtime row. */
  renderItem(item: T): ReactNode;
}
