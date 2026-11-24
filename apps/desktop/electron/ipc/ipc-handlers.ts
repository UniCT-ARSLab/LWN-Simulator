import { ipcMain } from "electron";
import Store from "electron-store";
import { pathToFileURL } from "url";
import {
  startBackend,
  stopBackend,
  waitForServer,
} from "../backend/backend-manager";

import {
  disconnectMainWindow,
  getLauncherPath,
  getMainWindow,
  setConnected,
} from "../window/window-manager";
import { validateRemote } from "../remote/remote-validator";

let handlersRegistered = false;

function isMainWindowSender(event: Electron.IpcMainInvokeEvent): boolean {
  return getMainWindow()?.webContents.id === event.sender.id;
}

function isLauncherSender(event: Electron.IpcMainInvokeEvent): boolean {
  return (
    isMainWindowSender(event) &&
    event.sender
      .getURL()
      .startsWith(pathToFileURL(getLauncherPath()).toString())
  );
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function registerIpcHandlers() {
  if (handlersRegistered) {
    return;
  }

  handlersRegistered = true;
  const store = new Store();

  ipcMain.handle("connect-local", async (event) => {
    if (!isLauncherSender(event)) {
      throw new Error("Only the launcher can start a connection");
    }

    const window = getMainWindow();

    if (!window) {
      return { success: false, message: "The main window is unavailable" };
    }

    try {
      const port = await startBackend();
      await waitForServer(port);
      await window.loadURL(`http://127.0.0.1:${port}`);
      setConnected(true);

      return { success: true };
    } catch (error) {
      stopBackend();
      setConnected(false);

      return {
        success: false,
        message: getErrorMessage(error),
      };
    }
  });

  ipcMain.handle("connect-remote", async (event, url: string) => {
    if (!isLauncherSender(event)) {
      throw new Error("Only the launcher can start a connection");
    }

    const window = getMainWindow();

    if (!window) {
      return { success: false, message: "The main window is unavailable" };
    }

    try {
      await validateRemote(url);

      await window.loadURL(url);

      setConnected(true);

      return {
        success: true,
      };
    } catch (err) {
      return {
        success: false,
        message: err instanceof Error ? err.message : String(err),
      };
    }
  });

  ipcMain.handle("disconnect", async (event) => {
    if (!isMainWindowSender(event)) {
      throw new Error("Only the main window can disconnect");
    }

    await disconnectMainWindow();
  });

  ipcMain.handle("sync-storage", (event, key: string, value: string) => {
    if (!isMainWindowSender(event)) {
      throw new Error("Only the main window can access storage");
    }

    store.set(key, value);
  });

  ipcMain.handle("get-storage", (event, key: string) => {
    if (!isMainWindowSender(event)) {
      throw new Error("Only the main window can access storage");
    }

    return store.get(key);
  });
}
