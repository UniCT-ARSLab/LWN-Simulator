"use client";

import { ConnectionForm } from "@/components";
import {
  Logo,
  LogoLayout,
  LogoSize,
  Spinner,
  SpinnerSize,
} from "@lwn-simulator/ui-components";
import { useState } from "react";

export default function Home() {
  // States
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [loadingText, setLoadingText] = useState<string>("");

  // Callbacks
  const handleLoadingChange = (nextLoadingText: string | null) => {
    setLoadingText(nextLoadingText ?? "");
    setIsLoading(nextLoadingText !== null);
  };

  return (
    <div className={`home-page page`}>
      {!isLoading ? (
        <>
          <Logo size={LogoSize.Lg} layout={LogoLayout.SecondaryBackground} />
          <h2 className="title">Choose a Connection</h2>
          <h3 className="sub-title">
            Select a local backend or connect to a remote server
          </h3>{" "}
          <ConnectionForm onLoadingChange={handleLoadingChange} />
        </>
      ) : (
        <>
          <Spinner size={SpinnerSize.Lg} />
          <p className="loading-text">{loadingText}</p>
        </>
      )}
    </div>
  );
}
