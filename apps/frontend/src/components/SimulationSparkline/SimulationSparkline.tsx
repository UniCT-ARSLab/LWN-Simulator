import { SimulationSparklineProps } from "./SimulationSparkline.types";

const SimulationSparkline = (props: SimulationSparklineProps) => {
  const values = props.values.length > 0 ? props.values : [0];
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const points = values.map((value, index) => {
    const x = values.length === 1 ? 50 : (index / (values.length - 1)) * 100;
    const y = 32 - ((value - min) / range) * 26;
    return `${x},${y}`;
  }).join(" ");

  return (
    <svg className="simulation-sparkline" viewBox="0 0 100 36" role="img" aria-label={props.label ?? "Simulation trend"} preserveAspectRatio="none">
      <polyline points={points} fill="none" vectorEffect="non-scaling-stroke" />
    </svg>
  );
};

export default SimulationSparkline;
