import {
  CreateProfileRequest,
  Profile,
  ProfileArchive,
  SimulationActivity,
  UpdateProfileRequest,
} from "@lwn-simulator/contracts";
import { PropsWithChildren } from "react";

/** Operations and state exposed by the profile service provider. */
export interface ProfileServiceContent {
  /** Profiles available to the current user. */
  profiles: Array<Profile>;

  /** Currently selected profile, or `null` while no profile is selected. */
  activeProfile: Profile | null;

  /** Identifier of the selected profile. */
  activeProfileID: string;

  /** Base URL scoped to the selected profile. */
  profileBaseUrl: string;

  /** Whether the initial profile list is being loaded. */
  loading: boolean;

  /** Last profile loading or mutation error. */
  error: Error | null;

  /** Selects a profile by identifier. */
  selectProfile: (id: string) => void;

  /** Creates a profile and selects it. */
  createProfile: (request: CreateProfileRequest) => Promise<Profile>;

  /** Updates a profile. */
  updateProfile: (request: UpdateProfileRequest) => Promise<Profile>;

  /** Deletes a profile. */
  deleteProfile: (id: string) => Promise<void>;

  /** Exports a profile as an archive. */
  exportProfile: (id?: string) => Promise<ProfileArchive>;

  /** Builds the download URL for a profile archive. */
  getExportProfileURL: (id?: string) => string;

  /** Imports an archive as a new profile and selects it. */
  importProfile: (archive: ProfileArchive) => Promise<Profile>;

  /** Shared simulation activity, regardless of the selected profile. */
  simulationActivity: SimulationActivity | null;

  /** Whether shared simulation activity is being loaded. */
  simulationActivityLoading: boolean;

  /** Applies a confirmed simulation status for the originating profile. */
  updateSimulationStatus: (profileID: string | undefined, status: SimulationActivity["status"]) => void;
}

/** Properties accepted by the profile service provider. */
export interface ProfileServiceProviderProps extends PropsWithChildren {
  /** Base URL of the backend API. */
  baseUrl: string;
}
