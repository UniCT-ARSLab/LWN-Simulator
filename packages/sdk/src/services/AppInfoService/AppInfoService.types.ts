import { AppInfoResponse, StatusResponse } from "@lwn-simulator/contracts";
import { PropsWithChildren } from "react";

/**
 * Represents the content exposed by the application information service.
 *
 * Provides methods for retrieving the application's current status and
 * application information from the configured backend.
 */
export interface AppInfoServiceContent {
  /**
   * Retrieves the current application status.
   *
   * @returns A promise resolving to the application's status information.
   */
  status: () => Promise<StatusResponse>;

  /**
   * Retrieves information about the application.
   *
   * @returns A promise resolving to the application's information.
   */
  appInfo: () => Promise<AppInfoResponse>;
}

/**
 * Properties for configuring the application information service provider.
 *
 * The provider makes application information service functionality available
 * to its descendant components.
 */
export interface AppInfoServiceProviderProps extends PropsWithChildren {
  /**
   * Base URL used to communicate with the application information service.
   */
  baseUrl: string;
}
