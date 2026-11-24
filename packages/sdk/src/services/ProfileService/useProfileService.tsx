import { useContext } from "react";
import ProfileServiceContext from "./ProfileServiceContext";

export const useProfileService = () => {
  const context = useContext(ProfileServiceContext);
  if (!context) {
    throw new Error(
      "useProfileService must be used inside `ProfileServiceProvider`",
    );
  }
  return context;
};
