import { useContext } from "react";
import GatewayServiceContext from "./GatewayServiceContext";

export const useGatewayService = () => {
  const context = useContext(GatewayServiceContext);
  if (!context) {
    throw new Error(
      "useGatewayService must be used inside `GatewayServiceProvider`",
    );
  }
  return context;
};
