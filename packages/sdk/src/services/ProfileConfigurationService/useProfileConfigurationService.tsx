import { useContext } from "react";
import ProfileConfigurationServiceContext from "./ProfileConfigurationServiceContext";

export const useProfileConfigurationService = () => {
  const context = useContext(ProfileConfigurationServiceContext);
  if (!context) {
    throw new Error(
      "useProfileConfigurationService must be used inside `ProfileConfigurationServiceProvider`",
    );
  }
  return context;
};
