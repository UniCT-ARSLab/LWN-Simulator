import cn from "classnames";
import { ProgressBar } from "@/components/common";
import { CodeEditorFormFieldProps } from "./CodeEditorFormField.types";

const CodeEditorFormField = (props: CodeEditorFormFieldProps) => {
  return (
    <div
      className={cn(
        "form-field code-editor-form-field",
        props.disabled && "disabled",
      )}
    >
      <textarea
        className="code-editor"
        id={props.name}
        name={props.name}
        value={props.value}
        placeholder={props.placeholder}
        spellCheck={false}
        disabled={props.disabled}
        readOnly={props.readOnly}
        aria-label={props.label}
        onChange={(event) => {
          let value = event.target.value;

          if (props.format) value = props.format(value);
          if (props.charsMax && value.length > props.charsMax) return;

          props.setValue(value);
        }}
        style={{ resize: props.resize ? "vertical" : "none" }}
      />
      {props.charsMax && (
        <div className="chars-check">
          <ProgressBar
            percentage={(props.value.length / props.charsMax) * 100}
          />
          <span>
            {props.value.length} / {props.charsMax} chars
          </span>
        </div>
      )}
    </div>
  );
};

export default CodeEditorFormField;
