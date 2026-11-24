import { createContext } from "react";
import { GatewayServiceContent } from "./GatewayService.types";

const GatewayServiceContext = createContext<GatewayServiceContent | null>(null);

export default GatewayServiceContext;
