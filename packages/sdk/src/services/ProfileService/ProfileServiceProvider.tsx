"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  CreateProfileRequest,
  Profile,
  ProfileArchive,
  RealtimeWebSocketMessage,
  SimulationActivity,
  UpdateProfileRequest,
} from "@lwn-simulator/contracts";
import { useWebSocketService } from "../WebSocketService";
import ProfileServiceContext from "./ProfileServiceContext";
import {
  ProfileServiceContent,
  ProfileServiceProviderProps,
} from "./ProfileService.types";
import { ProfileService } from "./ProfileService";
import {
  RealtimeWebSocketMessageType,
  SimulationStatus,
} from "../../models/RealtimeWebSocket";

const activeProfileStorageKey = "lwn-simulator.active-profile";
const defaultProfileID = "00000000-0000-4000-8000-000000000001";

const readStoredProfileID = (): string => {
  if (typeof window === "undefined") return defaultProfileID;
  try {
    return window.localStorage.getItem(activeProfileStorageKey) ?? defaultProfileID;
  } catch {
    return defaultProfileID;
  }
};

const ProfileServiceProvider = (props: ProfileServiceProviderProps) => {
  // States
  const [profiles, setProfiles] = useState<Array<Profile>>([]);
  const [activeProfileID, setActiveProfileID] = useState(readStoredProfileID);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [simulationActivity, setSimulationActivity] =
    useState<SimulationActivity | null>(null);
  const [simulationActivityLoading, setSimulationActivityLoading] =
    useState(true);

  // Hooks
  const profilesRef = useRef<Array<Profile>>([]);
  const simulationActivityRef = useRef<SimulationActivity | null>(null);
  const activeProfileIDRef = useRef(activeProfileID);
  const { connectionState, subscribe } = useWebSocketService();
  const websocketConnected = connectionState === "connected";

  // Memos
  const service = useMemo(
    () => new ProfileService(props.baseUrl),
    [props.baseUrl],
  );

  // Callbacks
  const commitProfiles = useCallback((nextProfiles: Array<Profile>) => {
    profilesRef.current = nextProfiles;
    setProfiles(nextProfiles);
  }, []);

  const commitSimulationActivity = useCallback(
    (nextActivity: SimulationActivity) => {
      simulationActivityRef.current = nextActivity;
      setSimulationActivity(nextActivity);
    },
    [],
  );

  const updateSimulationStatus = useCallback(
    (profileID: string | undefined, status: SimulationActivity["status"]) => {
      const active =
        status !== SimulationStatus.Idle &&
        status !== SimulationStatus.Stopped &&
        status !== SimulationStatus.Failed;
      const current = simulationActivityRef.current;
      if (active) {
        commitSimulationActivity({ active, profileID, status });
      } else if (
        current?.active &&
        (!current.profileID || current.profileID === profileID)
      ) {
        commitSimulationActivity({ active, status });
      }
    },
    [commitSimulationActivity],
  );

  const upsertProfile = useCallback(
    (nextProfile: Profile) => {
      const nextProfiles = profilesRef.current.some(
        (profile) => profile.id === nextProfile.id,
      )
        ? profilesRef.current.map((profile) =>
            profile.id === nextProfile.id ? nextProfile : profile,
          )
        : [...profilesRef.current, nextProfile];
      commitProfiles(nextProfiles);
    },
    [commitProfiles],
  );

  const persistActiveProfile = useCallback((id: string) => {
    activeProfileIDRef.current = id;
    setActiveProfileID(id);
    if (typeof window !== "undefined") {
      try {
        window.localStorage.setItem(activeProfileStorageKey, id);
      } catch {
        // The current session still owns its selection when storage is unavailable.
      }
    }
  }, []);

  const selectProfile = useCallback(
    (id: string) => {
      if (!profilesRef.current.some((profile) => profile.id === id)) return;
      persistActiveProfile(id);
    },
    [persistActiveProfile],
  );

  const getProfiles = useCallback(async (signal: AbortSignal) => {
    if (signal.aborted) return;
    setLoading(true);
    try {
      let previousProfiles: Array<Profile>;
      let nextProfiles: Array<Profile>;
      do {
        previousProfiles = profilesRef.current;
        nextProfiles = await service.getProfiles();
        if (signal.aborted) return;
      } while (previousProfiles !== profilesRef.current);
      commitProfiles(nextProfiles);
      const storedID = activeProfileIDRef.current;
      const nextActive = nextProfiles.some((profile) => profile.id === storedID)
        ? storedID
        : nextProfiles.find((profile) => profile.id === defaultProfileID)?.id ??
          nextProfiles[0]?.id ??
          "";
      if (nextActive) persistActiveProfile(nextActive);
      setError(null);
    } catch (loadError) {
      if (signal.aborted) return;
      setError(
        loadError instanceof Error
          ? loadError
          : new Error("Failed to load profiles"),
      );
    } finally {
      if (!signal.aborted) setLoading(false);
    }
  }, [commitProfiles, persistActiveProfile, service]);

  // Effects
  useEffect(() => {
    const request = new AbortController();
    void Promise.resolve().then(() => getProfiles(request.signal));
    return () => request.abort();
  }, [getProfiles, websocketConnected]);

  useEffect(() => {
    const previousActivity = simulationActivityRef.current;
    let cancelled = false;
    void service
      .getSimulationActivity()
      .then((activity) => {
        if (!cancelled && simulationActivityRef.current === previousActivity) {
          commitSimulationActivity(activity);
        }
      })
      .catch(() => {
        // Preserve the last confirmed activity while the backend is unavailable.
      })
      .finally(() => {
        if (!cancelled) setSimulationActivityLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [commitSimulationActivity, service, websocketConnected]);

  const handleRealtimeMessage = useCallback(
    (rawMessage: string) => {
      try {
        const message = JSON.parse(rawMessage) as RealtimeWebSocketMessage;
        if (
          (message.type === RealtimeWebSocketMessageType.ProfileCreated ||
            message.type === RealtimeWebSocketMessageType.ProfileUpdated) &&
          message.profile
        ) {
          const nextProfile = message.profile as Profile;
          upsertProfile(nextProfile);
        }
        if (
          message.type === RealtimeWebSocketMessageType.ProfileDeleted &&
          message.resourceID
        ) {
          const nextProfiles = profilesRef.current.filter(
            (profile) => profile.id !== message.resourceID,
          );
          commitProfiles(nextProfiles);
          if (
            activeProfileIDRef.current === message.resourceID &&
            nextProfiles[0]
          ) {
            persistActiveProfile(nextProfiles[0].id);
          }
        }
        if (
          message.type === RealtimeWebSocketMessageType.SimulationSnapshot &&
          message.snapshot
        ) {
          updateSimulationStatus(message.profileID, message.snapshot.state.status);
        }
      } catch {
        return;
      }
    },
    [
      commitProfiles,
      persistActiveProfile,
      updateSimulationStatus,
      upsertProfile,
    ],
  );

  useEffect(
    () => subscribe(handleRealtimeMessage),
    [handleRealtimeMessage, subscribe],
  );

  const createProfile = useCallback(
    async (request: CreateProfileRequest) => {
      try {
        const profile = await service.createProfile(request);
        upsertProfile(profile);
        persistActiveProfile(profile.id);
        setError(null);
        return profile;
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError
            : new Error("Failed to create profile"),
        );
        throw requestError;
      }
    },
    [persistActiveProfile, service, upsertProfile],
  );

  const updateProfile = useCallback(
    async (request: UpdateProfileRequest) => {
      try {
        const profile = await service.updateProfile(request);
        commitProfiles(
          profilesRef.current.map((current) =>
            current.id === profile.id ? profile : current,
          ),
        );
        setError(null);
        return profile;
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError
            : new Error("Failed to update profile"),
        );
        throw requestError;
      }
    },
    [commitProfiles, service],
  );

  const deleteProfile = useCallback(
    async (id: string) => {
      await service.deleteProfile(id);
      const nextProfiles = profilesRef.current.filter(
        (profile) => profile.id !== id,
      );
      commitProfiles(nextProfiles);
      if (activeProfileIDRef.current === id && nextProfiles[0]) {
        persistActiveProfile(nextProfiles[0].id);
      }
      setError(null);
    },
    [commitProfiles, persistActiveProfile, service],
  );

  const exportProfile = useCallback(
    (id = activeProfileIDRef.current): Promise<ProfileArchive> =>
      service.exportProfile(id),
    [service],
  );

  const getExportProfileURL = useCallback(
    (id = activeProfileIDRef.current): string =>
      service.getExportProfileURL(id),
    [service],
  );

  const importProfile = useCallback(
    async (archive: ProfileArchive) => {
      const profile = await service.importProfile(archive);
      upsertProfile(profile);
      persistActiveProfile(profile.id);
      setError(null);
      return profile;
    },
    [persistActiveProfile, service, upsertProfile],
  );

  const activeProfile =
    profiles.find((profile) => profile.id === activeProfileID) ?? null;

  // Memos
  const profileBaseUrl = `${props.baseUrl.replace(/\/+$/, "")}/profiles/${encodeURIComponent(
    activeProfileID,
  )}`;
  const value = useMemo(
    (): ProfileServiceContent => ({
      profiles,
      activeProfile,
      activeProfileID,
      profileBaseUrl,
      loading,
      error,
      selectProfile,
      createProfile,
      updateProfile,
      deleteProfile,
      exportProfile,
      getExportProfileURL,
      importProfile,
      simulationActivity,
      simulationActivityLoading,
      updateSimulationStatus,
    }),
    [
      profiles,
      activeProfile,
      activeProfileID,
      profileBaseUrl,
      loading,
      error,
      selectProfile,
      createProfile,
      updateProfile,
      deleteProfile,
      exportProfile,
      getExportProfileURL,
      importProfile,
      simulationActivity,
      simulationActivityLoading,
      updateSimulationStatus,
    ],
  );

  return (
    <ProfileServiceContext.Provider value={value}>
      {props.children}
    </ProfileServiceContext.Provider>
  );
};

export default ProfileServiceProvider;
