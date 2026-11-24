"use client";

import { createContext } from "react";
import { AppInfoServiceContent } from "./AppInfoService.types";

const AppInfoServiceContext = createContext<AppInfoServiceContent | null>(null);

export default AppInfoServiceContext;
