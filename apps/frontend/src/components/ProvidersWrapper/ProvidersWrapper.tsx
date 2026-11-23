"use client";

import { ThemeServiceProvider } from "@lwn-simulator/ui-components";
import { ProvidersWrapperProps } from "./ProvidersWrapper.types";
import {
  AppInfoServiceProvider,
  DeviceServiceProvider,
  GatewayServiceProvider,
  ProfileConfigurationServiceProvider,
  ProfileServiceProvider,
  useProfileService,
  SimulationServiceProvider,
  WebSocketServiceProvider,
} from "@lwn-simulator/sdk";
import Navbar from "../Navbar";
import Sidebar from "../Sidebar";
import Footer from "../Footer";
import { useEffect, useState } from "react";
import { ToastContainer } from "react-toastify";

interface ProfileScopedServicesProps {
  children: React.ReactNode;
}

const ProfileScopedServices = (props: ProfileScopedServicesProps) => {
  const profileService = useProfileService();

  return (
    <ProfileConfigurationServiceProvider
      key={profileService.activeProfileID}
      baseUrl={profileService.profileBaseUrl}
      profileID={profileService.activeProfileID}
    >
      <DeviceServiceProvider
        baseUrl={profileService.profileBaseUrl}
        profileID={profileService.activeProfileID}
      >
        <GatewayServiceProvider
          baseUrl={profileService.profileBaseUrl}
          profileID={profileService.activeProfileID}
        >
          <SimulationServiceProvider
            baseUrl={profileService.profileBaseUrl}
            profileID={profileService.activeProfileID}
            onControlStatus={profileService.updateSimulationStatus}
          >
            {props.children}
          </SimulationServiceProvider>
        </GatewayServiceProvider>
      </DeviceServiceProvider>
    </ProfileConfigurationServiceProvider>
  );
};

const ProvidersWrapper = (props: ProvidersWrapperProps) => {
  // States
  const [origin, setOrigin] = useState("http://localhost:8080/api");

  // Effects
  useEffect(() => {
    setOrigin(
      `${process.env.NODE_ENV === "development" ? "http://localhost:8080" : window.location.origin}/api`,
    );
  }, []);
  return (
      <ThemeServiceProvider>
        <AppInfoServiceProvider baseUrl={origin}>
          <WebSocketServiceProvider baseUrl={origin}>
            <ProfileServiceProvider baseUrl={origin}>
              <ProfileScopedServices>
                <Navbar />
                <Sidebar>
                  {props.children}
                  <Footer />
                </Sidebar>
                <ToastContainer />
              </ProfileScopedServices>
            </ProfileServiceProvider>
          </WebSocketServiceProvider>
        </AppInfoServiceProvider>
      </ThemeServiceProvider>
  );
};

export default ProvidersWrapper;
