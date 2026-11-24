"use client";

import {
  Button,
  ButtonType,
  Form,
  FormValue,
  NotificationHandler,
  Popup,
  selectField,
  textField,
  usePopup,
  useOutsideAlerter,
} from "@lwn-simulator/ui-components";
import { useProfileService } from "@lwn-simulator/sdk";
import { useMemo, useRef, useState } from "react";

type ProfileDialogMode = "create" | "rename" | null;
type ProfileDeleteTarget = {
  id: string;
  name: string;
};

const readFile = (file: File): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error("Unable to read the profile file"));
    reader.readAsText(file);
  });

const ProfileMenu = () => {
  const profileService = useProfileService();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [importing, setImporting] = useState(false);
  const [profileDialogMode, setProfileDialogMode] =
    useState<ProfileDialogMode>(null);
  const [profileDeleteTarget, setProfileDeleteTarget] =
    useState<ProfileDeleteTarget | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  useOutsideAlerter({
    ref: menuRef,
    onClickOutside: () => setOpen(false),
  });
  const profilePopup = usePopup({});
  const deletePopup = usePopup({});
  const notification = NotificationHandler.instance;
  const profileOptions = useMemo(
    () =>
      profileService.profiles.map((profile) => ({
        value: profile.id,
        displayed: <>{profile.name}</>,
      })),
    [profileService.profiles],
  );
  const profileFields = useMemo(
    () => [
      selectField({
        name: "profile",
        label: "Active profile",
        value: profileService.activeProfileID,
        placeholder: "Select a profile",
        error: null,
        options: profileOptions,
        onChange: (logic) => {
          const nextProfileID = logic.fieldsState.profile.value;
          if (typeof nextProfileID !== "string") return;
          profileService.selectProfile(nextProfileID);
          setOpen(false);
        },
      }),
    ],
    [profileOptions, profileService],
  );
  const profileFormKey = `${profileService.activeProfileID}:${profileService.profiles
    .map((profile) => `${profile.id}:${profile.name}`)
    .join("|")}`;

  const runAction = async (
    action: () => Promise<void>,
    success: string,
  ): Promise<boolean> => {
    setBusy(true);
    try {
      await action();
      notification.success(success);
      return true;
    } catch (error) {
      notification.error(
        error instanceof Error ? error.message : "The profile action failed",
      );
      return false;
    } finally {
      setBusy(false);
    }
  };

  const openCreateProfile = () => {
    setOpen(false);
    setProfileDialogMode("create");
    profilePopup.openPopup();
  };

  const openRenameProfile = () => {
    if (!profileService.activeProfile) return;
    setOpen(false);
    setProfileDialogMode("rename");
    profilePopup.openPopup();
  };

  const profileDialogFields = useMemo(() => {
    const isRename = profileDialogMode === "rename";
    return [
      textField({
        name: "name",
        label: "Profile name",
        value: isRename ? profileService.activeProfile?.name ?? "" : "",
        placeholder: "Enter a profile name",
        error: null,
        required: true,
      }),
    ];
  }, [profileDialogMode, profileService.activeProfile]);

  const submitProfileDialog = async (values: Record<string, FormValue>) => {
    const name = typeof values.name === "string" ? values.name.trim() : "";
    if (!name || !profileDialogMode) return;

    const succeeded = await runAction(
      async () => {
        if (profileDialogMode === "create") {
          await profileService.createProfile({ name });
          return;
        }
        await profileService.updateProfile({
          id: profileService.activeProfileID,
          name,
        });
      },
      profileDialogMode === "create" ? "Profile created" : "Profile renamed",
    );
    if (succeeded) {
      profilePopup.closePopup();
      setProfileDialogMode(null);
    }
  };

  const openDeleteConfirmation = () => {
    if (!profileService.activeProfile) return;
    setOpen(false);
    setProfileDeleteTarget({
      id: profileService.activeProfileID,
      name: profileService.activeProfile.name,
    });
    deletePopup.openPopup();
  };

  const confirmDeleteProfile = async () => {
    if (!profileDeleteTarget) return;
    const succeeded = await runAction(
      () => profileService.deleteProfile(profileDeleteTarget.id),
      "Profile deleted",
    );
    if (succeeded) {
      deletePopup.closePopup();
      setProfileDeleteTarget(null);
    }
  };

  const exportProfile = async () => {
    if (!profileService.activeProfile) return;
    await runAction(
      async () => {
        const response = await fetch(profileService.getExportProfileURL());
        if (!response.ok) {
          let message = "Unable to export the profile";
          try {
            const error = (await response.json()) as { error?: string };
            if (error.error) message = error.error;
          } catch {
            // Keep the generic message when the server response is not JSON.
          }
          throw new Error(message);
        }

        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement("a");
        anchor.href = url;
        anchor.download = `${profileService.activeProfile?.name ?? "lwn-profile"}.lwn-profile.json`;
        anchor.style.position = "fixed";
        anchor.style.left = "-9999px";
        document.body.appendChild(anchor);
        anchor.click();
        window.setTimeout(() => {
          anchor.remove();
          URL.revokeObjectURL(url);
        }, 1000);
      },
      "Profile exported",
    );
  };

  const importProfile = () => {
    fileInputRef.current?.click();
  };

  const handleFile = async (file: File | undefined) => {
    if (!file) return;
    setImporting(true);
    try {
      await runAction(
        async () => {
          const archive = JSON.parse(await readFile(file));
          await profileService.importProfile(archive);
        },
        "Profile imported",
      );
    } finally {
      setImporting(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  return (
    <>
      <div className="profile-menu" ref={menuRef}>
      <button
        className={`profile-menu-trigger${open ? " is-open" : ""}`}
        type="button"
        aria-expanded={open}
        aria-label="Select profile"
        onClick={() => setOpen((value) => !value)}
        disabled={busy || profileService.loading}
      >
        <span className="material-symbols-outlined icon">account_tree</span>
        <span className="profile-menu-name">
          {profileService.activeProfile?.name ?? "Profiles"}
        </span>
        <span className="material-symbols-outlined expand-icon">
          keyboard_arrow_down
        </span>
      </button>
      <div
        className={`profile-menu-panel${open ? " is-open" : ""}`}
        aria-hidden={!open}
      >
        <div className="profile-menu-selector">
          <Form
            key={profileFormKey}
            fields={profileFields}
            onSubmit={() => undefined}
            showSubmitButton={false}
          />
        </div>
        <div className="profile-menu-actions">
          <Button
            value="New profile"
            type={ButtonType.Outlined}
            onClick={openCreateProfile}
            disabled={busy}
          />
          <Button
            value="Rename"
            type={ButtonType.Outlined}
            onClick={openRenameProfile}
            disabled={busy || !profileService.activeProfile}
          />
          <Button
            value="Delete"
            type={ButtonType.Outlined}
            onClick={openDeleteConfirmation}
            disabled={busy || profileService.profiles.length <= 1}
          />
          <Button
            value="Export"
            type={ButtonType.Outlined}
            onClick={exportProfile}
            disabled={busy || !profileService.activeProfile}
          />
          <Button
            value="Import"
            type={ButtonType.Outlined}
            onClick={importProfile}
            disabled={busy || importing}
            loading={importing}
          />
        </div>
        <input
          ref={fileInputRef}
          className="profile-menu-file"
          type="file"
          accept="application/json,.json"
          onChange={(event) => void handleFile(event.target.files?.[0])}
        />
      </div>
      </div>
      <Popup
        logic={profilePopup}
        title={profileDialogMode === "rename" ? "Rename profile" : "New profile"}
      >
        <Form
          key={`${profileDialogMode}:${profileService.activeProfileID}`}
          fields={profileDialogFields}
          onSubmit={submitProfileDialog}
          submitButton={{
            value:
              profileDialogMode === "rename"
                ? "Rename profile"
                : "Create profile",
            disabled: busy,
          }}
        />
      </Popup>
      <Popup logic={deletePopup} title="Delete profile">
        <div className="profile-delete-dialog">
          <p>
            Delete profile <strong>{profileDeleteTarget?.name}</strong>? This
            also removes its devices, gateways and logs.
          </p>
          <div className="profile-delete-dialog-actions">
            <Button
              value="Cancel"
              type={ButtonType.Outlined}
              onClick={deletePopup.closePopup}
              disabled={busy}
            />
            <Button
              value="Delete profile"
              onClick={confirmDeleteProfile}
              disabled={busy || !profileDeleteTarget}
              loading={busy}
            />
          </div>
        </div>
      </Popup>
    </>
  );
};

export default ProfileMenu;
