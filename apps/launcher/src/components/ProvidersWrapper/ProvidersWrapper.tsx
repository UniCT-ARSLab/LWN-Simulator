"use client";

import { ThemeServiceProvider } from "@lwn-simulator/ui-components";
import { ProvidersWrapperProps } from "./ProvidersWrapper.types";
import { AppInfoServiceProvider } from "@lwn-simulator/sdk";
import { useEffect, useState } from "react";
import { ToastContainer } from "react-toastify";

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
        <main>{props.children}</main>
        <ToastContainer />
      </AppInfoServiceProvider>
    </ThemeServiceProvider>
  );
};

export default ProvidersWrapper;
