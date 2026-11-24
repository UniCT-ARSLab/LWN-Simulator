"use client";

import { createContext } from "react";
import { ProfileServiceContent } from "./ProfileService.types";

const ProfileServiceContext = createContext<ProfileServiceContent | null>(null);

export default ProfileServiceContext;
