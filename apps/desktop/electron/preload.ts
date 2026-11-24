import { contextBridge, ipcRenderer } from "electron";

async function connectLocal() {
  return ipcRenderer.invoke("connect-local");
}

async function connectRemote(url: string) {
  if (!/^https?:\/\//i.test(url)) {
    url = `http://${url}`;
  }

  return ipcRenderer.invoke("connect-remote", new URL(url).toString());
}

async function disconnect() {
  return ipcRenderer.invoke("disconnect");
}

async function syncStorage(key: string, value: string) {
  return ipcRenderer.invoke("sync-storage", key, value);
}

async function getStorage(key: string) {
  return ipcRenderer.invoke("get-storage", key);
}

contextBridge.exposeInMainWorld("electron", {
  connectLocal,
  connectRemote,
  disconnect,
  syncStorage,
  getStorage,
});
