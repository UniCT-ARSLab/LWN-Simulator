import { PropsWithChildren } from "react";

/**
 * Defines the available layout variants for a card component.
 */
export enum CardLayout {
  /** Default card layout without additional spacing adjustments. */
  Default = "l-default",

  /** Card layout with internal padding applied to its content. */
  Padded = "l-padded",
}

/**
 * Defines the visual style variants for a card component.
 */
export enum CardType {
  /** Standard card style with the default appearance. */
  Default = "t-default",

  /** Transparent card style without a background. */
  Transparent = "t-transparent",
}

/**
 * Properties for configuring a card component.
 */
export interface CardProps extends PropsWithChildren {
  /**
   * Additional CSS class names applied to the card element.
   */
  className?: string;

  /**
   * Layout variant that controls the card's internal structure and spacing.
   *
   * @default CardLayout.Default
   */
  layout?: CardLayout;

  /**
   * Visual style variant applied to the card.
   *
   * @default CardType.Default
   */
  type?: CardType;
}
