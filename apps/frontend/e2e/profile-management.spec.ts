import { expect, test } from "@playwright/test";
import { mockBackend } from "./support/mockBackend";

test("manages a profile through create, rename, export, import and delete", async ({
  page,
}) => {
  await mockBackend(page);
  await page.goto("/hardware/devices");

  const profileTrigger = page.getByRole("button", { name: "Select profile" });
  await profileTrigger.click();
  await page.getByRole("button", { name: "New profile" }).click();
  await page.getByLabel("Profile name").fill("Catania demo");
  await page.getByRole("button", { name: "Create profile" }).click();
  await expect(page.locator(".profile-menu-name")).toHaveText("Catania demo");

  await profileTrigger.click();
  await page.getByRole("button", { name: "Rename" }).click();
  await page.getByLabel("Profile name").fill("Catania renamed");
  await page.getByRole("button", { name: "Rename profile" }).click();
  await expect(page.locator(".profile-menu-name")).toHaveText("Catania renamed");

  await profileTrigger.click();
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(
    /^Catania(?: |_)renamed\.lwn-profile\.json$/,
  );

  await page.locator(".profile-menu-file").setInputFiles({
    name: "imported-profile.json",
    mimeType: "application/json",
    buffer: Buffer.from(
      JSON.stringify({
        format: "lwn-profile",
        version: 1,
        profile: { id: "external-profile", name: "Imported profile" },
        devices: [],
        gateways: [],
        logs: [],
      }),
    ),
  });
  await expect(page.locator(".profile-menu-name")).toHaveText("Imported profile");

  await profileTrigger.click();
  await page.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator(".popup-header strong")).toHaveText("Delete profile");
  await page.getByRole("button", { name: "Delete profile" }).click();
  await expect(page.locator(".profile-menu-name")).toHaveText("Default profile");
});

test("waits for the available profiles before loading hardware from a stale selection", async ({ page }) => {
  await mockBackend(page);
  await page.addInitScript(() => localStorage.setItem("lwn-simulator.active-profile", "deleted-profile"));
  const requestedProfiles = new Set<string>();
  page.on("request", request => {
    const profileID = new URL(request.url()).pathname.match(/\/api\/profiles\/([^/]+)\//)?.[1];
    if (profileID) requestedProfiles.add(profileID);
  });
  await page.goto("/hardware/devices");
  await expect.poll(() => requestedProfiles.has("profile-1")).toBe(true);
  expect(requestedProfiles.has("deleted-profile")).toBe(false);
});
