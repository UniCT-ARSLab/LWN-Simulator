# Frontend test support

This directory contains shared test infrastructure rather than feature tests.

- `fixtures/` contains stable identifiers and reusable test data.
- `mocks/handlers.ts` and `mocks/server.ts` provide the MSW HTTP boundary.
- `mocks/webSocket.ts` provides a controllable WebSocket test double.

Feature tests should live next to the component, hook, provider, or page they
exercise. End-to-end journeys belong in `apps/frontend/e2e/`.
