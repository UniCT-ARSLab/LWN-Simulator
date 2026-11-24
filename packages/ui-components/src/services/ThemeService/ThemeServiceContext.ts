"use client";

import { createContext } from "react";
import { Theme, ThemeServiceContent } from "./ThemeService.types";

export const ThemeServiceContext = createContext<ThemeServiceContent>({
  theme: Theme.Dark,
  setTheme: () => {},
});
