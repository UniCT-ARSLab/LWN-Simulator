import { ProgressBarProps, ProgressBarType } from "./ProgressBar.types";

const ProgressBar = (props: ProgressBarProps) => {
  const type = props.type || ProgressBarType.Primary;

  return (
    <div className="progress-bar">
      <div
        className="progress"
        style={{ width: `${props.percentage}%`, background: `var(${type})` }}
      />
    </div>
  );
};

export default ProgressBar;
