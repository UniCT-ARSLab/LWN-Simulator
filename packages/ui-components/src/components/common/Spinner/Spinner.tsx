import { CSSProperties } from "react";
import { SpinnerProps, SpinnerSize, SpinnerType } from "./Spinner.types";

const Spinner = (props: SpinnerProps) => {
  const size = props.size || SpinnerSize.Md;
  const type = props.type || SpinnerType.Primary;

  return (
    <span
      style={{ "--size": size, "--color-2": `var(${type})` } as CSSProperties}
      className="spinner"
    ></span>
  );
};

export default Spinner;
