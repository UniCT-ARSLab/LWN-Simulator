import { AppInfoResponse, StatusResponse } from "@lwn-simulator/contracts";
import { BaseService } from "../../models";

/**
 * Service responsible for retrieving application status and information
 * from the backend.
 *
 * Uses the application information API namespace configured by the base
 * service instance.
 */
export class AppInfoService extends BaseService {
  constructor(baseUrl: string) {
    super("app-info", baseUrl);
  }

  /**
   * Retrieves the current application status from the backend.
   *
   * @returns A promise resolving to the application's status information.
   */
  public status = async (): Promise<StatusResponse> => {
    const res: StatusResponse = await this.apiCaller.get("/status");

    return res;
  };

  /**
   * Retrieves information about the application from the backend.
   *
   * @returns A promise resolving to the application's information.
   */
  public appInfo = async (): Promise<AppInfoResponse> => {
    const res: AppInfoResponse = await this.apiCaller.get("/info");

    return res;
  };
}
