# Simulation engine

The engine is the deterministic Go core of LWN Simulator. It advances
simulation time, schedules protocol work, maintains runtime devices and
gateways, models radio outcomes and records the events needed by the backend
and log viewer.

It can run with a real clock for normal simulations or with a manual clock
for deterministic tests. An optional seed makes radio and scheduling outcomes
repeatable.

## Responsibilities

- simulation lifecycle: start, pause, resume and stop;
- priority-based event scheduling and clock control;
- device and gateway runtime registration;
- LoRaWAN 1.0.x sessions, OTAA, ABP, frame counters and receive windows;
- Class A, B and C behavior, confirmed traffic, acknowledgements and retries;
- regional channels, airtime, RSSI, SNR, coverage, collisions and packet loss;
- virtual gateway traffic and real Semtech UDP gateway adapters;
- snapshots, metrics, checkpoints and ordered event logs.

The engine is intentionally focused on behavior that affects simulation
results. It does not attempt to reproduce every internal detail of physical
radio hardware when that detail cannot change the observable network result.

## Structure

- `engine.go`, `errors.go`, `registry.go`, `scheduler.go` and
  `validation.go`: stable public facades;
- `internal/runtime/`: lifecycle, protocol sessions and traffic orchestration;
- `gateway/`: real-gateway transport adapters;
- `lorawan/`: frame security, parsing and MAC-command support;
- `regional/`: regional plans, channels, data rates and payload limits;
- `geometry/`: distance and coverage calculations;
- `scheduler/`, `registry/` and `validation/`: reusable runtime services;
- `types/`: clocks, scheduler events and runtime sink abstractions;
- `cmd/udp-gateway-simulator/`: external-style UDP gateway simulator utility.

## Use and test

The backend consumes the engine through its public Go package. Engine tests
are colocated with each domain package and include deterministic manual-clock
scenarios and race-enabled runs.

```bash
cd packages/engine
go test ./...
go test -race ./...
```

The engine is also included in the root checks through `yarn test` and
`yarn check`.
