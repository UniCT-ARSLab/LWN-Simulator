import { app, session } from "electron";
import { registerIpcHandlers } from "./ipc";
import { installLinuxDesktopEntry } from "./system";
import { createMainWindow, getMainWindow } from "./window";
import { stopBackend } from "./backend";

const isGeolocationPermission = (permission: string): boolean =>
  permission === "geolocation" || permission === "geolocation-approximate";

const hasSingleInstanceLock = app.requestSingleInstanceLock();

if (!hasSingleInstanceLock) {
  app.quit();
} else {
  app.on("second-instance", () => {
    const window = getMainWindow();

    if (!window) {
      return;
    }

    if (window.isMinimized()) {
      window.restore();
    }

    window.focus();
  });

  app.whenReady().then(() => {
    session.defaultSession.setPermissionCheckHandler(
      (_webContents, permission) => isGeolocationPermission(permission),
    );

    session.defaultSession.setPermissionRequestHandler(
      (_webContents, permission, callback) => {
        callback(isGeolocationPermission(permission));
      },
    );

    registerIpcHandlers();

    if (app.isPackaged) {
      installLinuxDesktopEntry();
    }

    createMainWindow();
  });

  app.on("activate", () => {
    if (!getMainWindow()) {
      createMainWindow();
    }
  });

  app.on("before-quit", () => {
    stopBackend();
  });
}

app.on("window-all-closed", () => {
  stopBackend();

  if (process.platform !== "darwin") {
    app.quit();
  }
});
