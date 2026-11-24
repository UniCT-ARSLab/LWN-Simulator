"use client";

import { createContext } from "react";
import { ProfileConfigurationServiceContent } from "./ProfileConfigurationService.types";

const ProfileConfigurationServiceContext =
  createContext<ProfileConfigurationServiceContent | null>(null);

export default ProfileConfigurationServiceContext;
