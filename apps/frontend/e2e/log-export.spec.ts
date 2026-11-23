import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { mockBackend } from "./support/mockBackend";

test("downloads all logs as a ZIP archive", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/logs");
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export all (ZIP)" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("simulation-logs.zip");
  const path = await download.path();
  expect(await readFile(path!, "utf8")).toBe("PK-test-archive");
  await expect(page).toHaveURL(/\/logs\/?$/);
});

test("downloads the selected run as JSON", async ({ page }) => {
  await mockBackend(page);
  await page.goto("/logs/detail?runId=run-new");
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export JSON" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("simulation-run-new.json");
  const path = await download.path();
  const exported = JSON.parse(await readFile(path!, "utf8"));
  expect(exported.run.summary.id).toBe("run-new");
  expect(exported.events[0].message).toBe("Exported uplink");
  await expect(page).toHaveURL(/runId=run-new/);
});

test("disables the archive button when there are no logs", async ({ page }) => {
  await mockBackend(page, { logs: "empty" });
  await page.goto("/logs");
  await expect(page.getByRole("button", { name: "Export all (ZIP)" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Export telemetry summary" })).toBeDisabled();
});

for (const scenario of [
  { path: "/logs", button: "Export telemetry summary", filename: "simulation-telemetry-summary.csv", content: "run_id,packet_success_rate_pct\nrun-new,80\n" },
  { path: "/logs/detail?runId=run-new", button: "Export telemetry", filename: "simulation-run-new-telemetry.zip", content: "PK-telemetry-archive" },
]) {
  test(`downloads ${scenario.filename} from the logs UI`, async ({ page }) => {
    await mockBackend(page);
    await page.goto(scenario.path);
    const pendingDownload = page.waitForEvent("download");
    await page.getByRole("button", { name: scenario.button, exact: true }).click();
    const download = await pendingDownload;
    expect(download.suggestedFilename()).toBe(scenario.filename);
    expect(await readFile((await download.path())!, "utf8")).toBe(scenario.content);
  });
}
