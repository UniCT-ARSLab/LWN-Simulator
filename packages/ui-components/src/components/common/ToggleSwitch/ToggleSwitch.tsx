import { ToggleSwitchProps, ToggleSwitchType } from "./ToggleSwitch.types";
import cn from "classnames";

const ToggleSwitch = (props: ToggleSwitchProps) => {
  const type = props.type || ToggleSwitchType.Boolean;

  return (
    <div
      className={cn("toggle-switch", props.value && "active", type)}
      onClick={() => {
        props.onToggle(!props.value);
      }}
    >
      {type == ToggleSwitchType.Choose && (
        <div className="false-icon">{props.falseIcon}</div>
      )}
      {type == ToggleSwitchType.Choose && (
        <div className="true-icon">{props.trueIcon}</div>
      )}
      <div className="trigger">
        {type == ToggleSwitchType.Boolean &&
          (props.value ? props.trueIcon : props.falseIcon)}
      </div>
    </div>
  );
};

export default ToggleSwitch;
