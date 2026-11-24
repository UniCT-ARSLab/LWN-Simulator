# SDK

The SDK is the TypeScript client layer shared by the frontend and desktop
launcher. It centralizes HTTP requests, WebSocket connections, typed API
models and React providers so every client observes the same backend behavior.

## Responsibilities

- call the generated profile, device, gateway and simulation APIs;
- manage WebSocket connection state, reconnection and live messages;
- expose profile-scoped services to React applications;
- keep transport details out of pages and feature components;
- provide typed errors and request results based on the shared contracts.

## Structure

- `src/services/`: service singletons, types, contexts, providers and hooks;
- `src/models/`: transport and API client models;
- `src/components/`: SDK-level provider composition components;
- `src/hooks/`: shared React hooks;
- `src/utils/`: transport and client utilities;
- `src/index.ts`: public package exports.

Each service follows the same structure: a `{Name}Service` singleton,
`{Name}Service.types.ts`, context, provider, hook and `index.ts` exports. The
main services cover application information, devices, gateways, profiles,
profile configuration, simulations and WebSockets.

## Build and lint

```bash
yarn workspace @lwn-simulator/sdk build
yarn workspace @lwn-simulator/sdk lint
```

The SDK depends on generated models from `@lwn-simulator/contracts`; run
`yarn generate` first when the OpenAPI schemas change.
