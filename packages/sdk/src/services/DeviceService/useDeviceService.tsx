import { useContext } from "react";
import DeviceServiceContext from "./DeviceServiceContext";

export const useDeviceService = () => {
  const context = useContext(DeviceServiceContext);

  if (!context) {
    throw new Error(
      "useDeviceService must be used inside `DeviceServiceProvider`",
    );
  }

  return context;
};
