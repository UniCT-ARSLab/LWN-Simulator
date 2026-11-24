"use client";

import { useContext } from "react";
import { ThemeServiceContext } from "./ThemeServiceContext";

export const useThemeService = () => {
  const context = useContext(ThemeServiceContext);

  if (context === undefined) {
    throw new Error(
      "useThemeService must be used inside `ThemeServiceProvider`",
    );
  }

  return context;
};
