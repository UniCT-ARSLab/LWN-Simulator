import { PropsWithChildren } from "react";

/**
 * Defines the available application themes.
 */
export enum Theme {
  /** Light application theme. */
  Light = "light",

  /** Dark application theme. */
  Dark = "dark",
}

/**
 * Content exposed by the theme service.
 *
 * Provides the current application theme and a function to change it.
 */
export interface ThemeServiceContent {
  /**
   * Currently active application theme.
   */
  theme: Theme;

  /**
   * Changes the active application theme.
   *
   * @param theme Theme to apply.
   */
  setTheme: (theme: Theme) => void;
}

/** Properties accepted by the theme service provider. */
export type ThemeServiceProviderProps = PropsWithChildren;
