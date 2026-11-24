import { BrowserWindow, app, shell } from "electron";
import path from "path";
import { stopBackend } from "../backend";
import { createMenu } from "./window-menu";

let mainWindow: BrowserWindow | undefined;
let disconnectItem: Electron.MenuItem | null = null;

export function createMainWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 1280,
    minHeight: 800,
    webPreferences: {
      preload: path.join(__dirname, "..", "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
    },
  });

  disconnectItem = createMenu(disconnectMainWindow);

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);

    return {
      action: "deny",
    };
  });

  mainWindow.on("closed", () => {
    mainWindow = undefined;
    disconnectItem = null;
  });

  mainWindow.loadFile(getLauncherPath());
}

export function setConnected(value: boolean) {
  if (disconnectItem) {
    disconnectItem.enabled = value;
  }
}

export function getMainWindow() {
  return mainWindow;
}

export async function disconnectMainWindow(): Promise<void> {
  stopBackend();
  setConnected(false);

  if (mainWindow) {
    await mainWindow.loadFile(getLauncherPath());
  }
}

export function getLauncherPath() {
  if (app.isPackaged) {
    return path.join(process.resourcesPath, "launcher", "out", "index.html");
  }

  return path.join(app.getAppPath(), "..", "launcher", "out", "index.html");
}
