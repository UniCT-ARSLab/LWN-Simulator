import {
  CreateProfileRequest,
  CreateProfileResponse,
  GetProfileResponse,
  GetProfilesResponse,
  ImportProfileRequest,
  ImportProfileResponse,
  Profile,
  ProfileArchive,
  SimulationActivity,
  UpdateProfileRequest,
  UpdateProfileResponse,
} from "@lwn-simulator/contracts";
import { BaseService } from "../../models";
import { ApiCaller } from "../../models/ApiCaller";

export class ProfileService extends BaseService {
  private readonly simulationApiCaller: ApiCaller;

  constructor(baseUrl: string) {
    super("profile", baseUrl);
    this.simulationApiCaller = new ApiCaller("simulation", baseUrl);
  }

  public getProfiles = async (): Promise<Array<Profile>> => {
    const response: GetProfilesResponse = await this.apiCaller.get(
      "/get-profiles",
    );
    return response.profiles;
  };

  public getProfile = async (id: string): Promise<Profile> => {
    const response: GetProfileResponse = await this.apiCaller.get(
      `/get-profile/${encodeURIComponent(id)}`,
    );
    return response.profile;
  };

  public createProfile = async (request: CreateProfileRequest): Promise<Profile> => {
    const response: CreateProfileResponse = await this.apiCaller.post(
      "/create-profile",
      request,
    );
    return response.profile;
  };

  public updateProfile = async (request: UpdateProfileRequest): Promise<Profile> => {
    const response: UpdateProfileResponse = await this.apiCaller.put(
      `/update-profile/${encodeURIComponent(request.id)}`,
      request,
    );
    return response.profile;
  };

  public deleteProfile = async (id: string): Promise<void> => {
    await this.apiCaller.delete(`/delete-profile/${encodeURIComponent(id)}`);
  };

  public exportProfile = async (id: string): Promise<ProfileArchive> =>
    this.apiCaller.get<ProfileArchive>(
      `/export-profile/${encodeURIComponent(id)}`,
      { timeout: 120000 },
    );

  public getExportProfileURL = (id: string): string =>
    `${this.apiCaller.buildUrl(
      `/export-profile/${encodeURIComponent(id)}`,
    )}?download=1`;

  public importProfile = async (archive: ProfileArchive): Promise<Profile> => {
    const request: ImportProfileRequest = { archive };
    const response: ImportProfileResponse = await this.apiCaller.post(
      "/import-profile",
      request,
    );
    return response.profile;
  };

  public getSimulationActivity = async (): Promise<SimulationActivity> =>
    this.simulationApiCaller.get<SimulationActivity>("/activity");
}
