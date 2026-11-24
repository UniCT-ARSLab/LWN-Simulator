import { PropsWithChildren } from "react";

/**
 * Properties for configuring a page header component.
 *
 * The page header displays the page title and optional subtitle, while
 * allowing additional actions or controls to be rendered alongside the header
 * content.
 */
export interface PageHeaderProps extends PropsWithChildren {
  /**
   * Main title displayed in the page header.
   */
  title: string;

  /**
   * Additional descriptive text displayed below the title.
   */
  subTitle?: string;

  /**
   * Additional content rendered on the right side of the page header.
   *
   * Typically used for actions or controls such as buttons, filters, or other
   * interactive elements.
   */
}
