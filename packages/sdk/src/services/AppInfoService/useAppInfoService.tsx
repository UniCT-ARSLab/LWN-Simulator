import { useContext } from "react";
import AppInfoServiceContext from "./AppInfoServiceContext";

export const useAppInfoService = () => {
  const context = useContext(AppInfoServiceContext);

  if (!context) {
    throw new Error(
      "useAppInfoService must be used inside `AppInfoServiceProvider`",
    );
  }

  return context;
};
