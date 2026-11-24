# Desktop launcher

The launcher is the connection screen shown before the desktop application
opens the simulator. It lets the user choose between starting a local backend
and connecting to an existing remote backend, then hands the selected
connection to the Electron shell.

## Structure

- `src/app/`: Next.js entry page and global launcher layout;
- `src/components/`: connection form and launcher-specific UI;
- `src/styles/`: launcher styles and responsive layout rules;
- `public/`: launcher assets.

The launcher uses the shared SDK for connection checks and the shared UI
components for the logo, fields, buttons and loading state.

## Development

```bash
yarn workspace @lwn-simulator/desktop-launcher dev
```

## Validation and build

```bash
yarn workspace @lwn-simulator/desktop-launcher lint
yarn workspace @lwn-simulator/desktop-launcher lint:css
yarn workspace @lwn-simulator/desktop-launcher build
```
