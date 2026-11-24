export interface ElectronAPI {
  connectLocal(): Promise<{ success: boolean; message?: string }>;
  connectRemote(url: string): Promise<{
    success: boolean;
    message?: string;
  }>;
  disconnect(): Promise<void>;
  getStorage(key: string): Promise<string | null>;
  syncStorage(key: string, value: string): Promise<void>;
}

declare global {
  interface Window {
    electron: ElectronAPI;
  }
}
