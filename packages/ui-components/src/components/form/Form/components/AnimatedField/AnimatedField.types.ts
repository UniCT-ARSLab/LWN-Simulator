import { FormField, FormLogic } from "../../Form.types";

/**
 * Properties for configuring an animated form field wrapper.
 *
 * Provides the field configuration, visibility state, and form logic required
 * to manage field rendering and transitions.
 */
export interface AnimatedFieldProps {
  /**
   * Configuration of the form field to render.
   */
  field: FormField;

  /**
   * Determines whether the field should be visible.
   *
   * Used to control the rendering state and animation transitions.
   */
  visible: boolean;

  /**
   * Form state and actions used to manage field values and behavior.
   */
  formLogic: FormLogic;
}
