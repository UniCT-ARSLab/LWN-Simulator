# Frontend

The frontend is the Next.js web application used to configure networks, run
simulations and inspect their results. It provides the main dashboard for
simulation control, hardware management, live maps, packet activity, profiles
and historical logs.

The interface uses the shared SDK for API and WebSocket communication and the
shared UI component package for common controls and visual patterns.

## Main areas

- `/simulation`: simulation dashboard and live runtime information;
- `/hardware`: device, gateway and gateway-bridge configuration;
- `/logs`: simulation history and per-run event inspection;
- profile controls in the navigation: create, select, import, export, rename
  and delete independent network configurations.

## Structure

- `src/app/`: route pages and layouts;
- `src/components/`: feature-specific visual components grouped by domain;
- `src/services/`: frontend providers built on top of the SDK;
- `src/styles/`: global and application-level styles;
- `src/utils/`: shared browser and presentation utilities;
- `public/`: static assets and map resources.

## Development

From the repository root:

```bash
yarn build:packages
yarn workspace @lwn-simulator/frontend dev
```

Or use `yarn dev` to start the frontend and backend together. The development
server normally runs at <http://localhost:3000>.

## Validation and build

```bash
yarn workspace @lwn-simulator/frontend lint:ts
yarn workspace @lwn-simulator/frontend lint:css
yarn workspace @lwn-simulator/frontend build
```

The build also prepares the frontend for the Electron distribution when it is
invoked through the desktop build scripts.
