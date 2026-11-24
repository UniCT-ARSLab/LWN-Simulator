const storage = {
  get: (key: string): Promise<string | null> => {
    if (window.electron) {
      return window.electron.getStorage(key);
    }

    return Promise.resolve(localStorage.getItem(key));
  },

  set: async (key: string, value: string): Promise<void> => {
    localStorage.setItem(key, value);

    if (window.electron) {
      await window.electron.syncStorage(key, value);
    }
  },
};

export default storage;
