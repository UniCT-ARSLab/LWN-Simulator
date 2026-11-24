"use client";

import {
  Button,
  ButtonType,
  Card,
  CardLayout,
  Form,
} from "@lwn-simulator/ui-components";
import { SimulationStatus } from "@lwn-simulator/contracts";
import { SimulationDashboardControlsProps } from "./SimulationDashboardControls.types";

export const SimulationDashboardControls = (
  props: SimulationDashboardControlsProps,
) => {
  const isRunning = props.status === SimulationStatus.Running;
  const isPaused = props.status === SimulationStatus.Paused;

  return (
    <Card
      className="simulation-dashboard__controls simulation-dashboard-controls"
      layout={CardLayout.Padded}
    >
      <Form
        fields={props.fields}
        onSubmit={props.onStart}
        submitButton={{
          value: "Start simulation",
          className: "simulation-dashboard__start-button",
          disabled: !props.canStart,
        }}
      />
      {props.requiresRealtime && (
        <p className="simulation-dashboard__simulation-lock" role="status">
          External gateway connections use real time. Simulation speed is fixed at 1x.
        </p>
      )}
      {props.hardwareLoaded && !props.hasActiveHardware && !isRunning && !isPaused && (
        <p className="simulation-dashboard__simulation-lock" role="status">
          Add at least one active device or gateway before starting a simulation.
        </p>
      )}
      {props.activeSimulationIsElsewhere && (
        <p className="simulation-dashboard__simulation-lock" role="status">
          {props.activeSimulationProfileName
            ? `A simulation is active in ${props.activeSimulationProfileName}. `
            : "A simulation is active in another profile. "}
          Only one simulation can run at a time. Stop it before starting a new
          one.
        </p>
      )}
      {(isRunning || isPaused) && (
        <div className="simulation-dashboard__control-actions">
          {isRunning && (
            <Button
              value="Pause"
              type={ButtonType.Outlined}
              onClick={props.onPause}
            />
          )}
          {isPaused && <Button value="Resume" onClick={props.onResume} />}
          <Button
            value="Stop"
            type={ButtonType.Outlined}
            onClick={props.onStop}
          />
        </div>
      )}
    </Card>
  );
};
