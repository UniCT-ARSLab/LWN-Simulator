import { createContext } from "react";
import { SimulationServiceContent } from "./SimulationService.types";

const SimulationServiceContext = createContext<SimulationServiceContent | null>(
  null,
);

export default SimulationServiceContext;
