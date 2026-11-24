# Desktop application

The desktop application is the Electron distribution of LWN Simulator. It
packages the web interface with a local backend option, provides the native
window and system integration, and supports connecting to a remote backend
through the launcher.

## Responsibilities

- create and manage the main Electron window;
- start and stop the bundled Go backend when using local mode;
- expose the small IPC bridge required by the renderer;
- validate remote connection settings;
- provide platform-specific packaging and desktop integration.

## Structure

- `electron/main.ts`: Electron application lifecycle and permissions;
- `electron/backend/`: local backend process management and port handling;
- `electron/ipc/`: renderer-to-main process handlers;
- `electron/remote/`: remote backend connection validation;
- `electron/system/`: operating-system integration;
- `electron/window/`: window creation and menu management;
- `assets/`: bundled backend resources;
- `public/`: application icons and static assets.

## Development

```bash
yarn dev:electron
```

This builds the shared packages and frontend, builds the local backend and
starts Electron. The development backend uses the data in
`apps/backend/dev-db/`.

## Build and lint

```bash
yarn workspace @lwn-simulator/desktop compile
yarn workspace @lwn-simulator/desktop lint
yarn workspace @lwn-simulator/desktop build
```

The root distribution scripts produce platform-specific artifacts for Linux,
macOS and Windows. See the [distribution documentation](https://giuseppe-compagnone.github.io/LWN-Simulator/docs/en/getting-started/desktop-edition)
for user-facing installation instructions.
