# Contracts

The contracts package is the shared API boundary of LWN Simulator. It defines
the OpenAPI schemas used to validate requests and responses and generates the
TypeScript and Go models consumed by the frontend, SDK and backend.

Keeping the schemas in one package prevents the clients and server from
drifting apart as profiles, hardware, simulations, logs and WebSocket messages
evolve.

## Structure

- `openapi.yaml`: root OpenAPI document;
- `schemas/`: domain schemas grouped by Device, Gateway, Profile, Simulation
  and Realtime;
- `scripts/`: model-generation helpers;
- `generated/`: generated TypeScript and Go output.

Generated files should not be edited by hand. Change the source schema and
regenerate the package instead.

## Generate models

From the repository root:

```bash
yarn generate
```

Or from this package:

```bash
yarn workspace @lwn-simulator/contracts build
```

The build bundles the OpenAPI document, generates TypeScript models, generates
Go models and prepares the public model exports.
