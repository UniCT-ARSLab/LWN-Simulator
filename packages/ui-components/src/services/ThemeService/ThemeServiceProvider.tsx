"use client";

import { useEffect, useState } from "react";
import { Theme, ThemeServiceProviderProps } from "./ThemeService.types";
import { ThemeServiceContext } from "./ThemeServiceContext";
import cn from "classnames";
import storage from "@/utils/storage";
import { ThemeService } from "./ThemeService";

const ThemeServiceProvider = (props: ThemeServiceProviderProps) => {
  // States
  const [theme, setTheme] = useState<Theme>(Theme.Dark);

  // Callbacks
  const setDefaultTheme = async () => {
    const defaultTheme = await ThemeService.instance.getDefaultTheme();

    setTheme(defaultTheme);
  };

  // Effects
  useEffect(() => {
    setDefaultTheme();
  }, []);

  useEffect(() => {
    document.body.className = cn(
      document.body.className.replace(/(dark|light)-theme/, "").trim(),
      `${theme}-theme`,
    );
    storage.set("theme", theme);
  }, [theme]);

  return (
    <ThemeServiceContext.Provider value={{ theme, setTheme }}>
      {props.children}
    </ThemeServiceContext.Provider>
  );
};

export default ThemeServiceProvider;
