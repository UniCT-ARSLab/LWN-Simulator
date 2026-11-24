import { useContext } from "react";
import SimulationServiceContext from "./SimulationServiceContext";

export const useSimulationService = () => {
  const context = useContext(SimulationServiceContext);

  if (!context) {
    throw new Error(
      "useSimulationService must be used inside `SimulationServiceProvider`",
    );
  }

  return context;
};
