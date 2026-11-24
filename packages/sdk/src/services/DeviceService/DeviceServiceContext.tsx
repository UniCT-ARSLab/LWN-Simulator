import { createContext } from "react";
import { DeviceServiceContent } from "./DeviceService.types";

const DeviceServiceContext = createContext<DeviceServiceContent | null>(null);

export default DeviceServiceContext;
