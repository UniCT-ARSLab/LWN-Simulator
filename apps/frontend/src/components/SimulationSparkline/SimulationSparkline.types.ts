/** Properties used to render a compact trend sparkline. */
export interface SimulationSparklineProps {
  /** Numeric values plotted from oldest to newest. */
  values: Array<number>;
  /** Accessible label describing the trend. */
  label?: string;
}
