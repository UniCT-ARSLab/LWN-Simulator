import { ReactNode } from "react";

/** Grid layouts available for an entity-details section. */
export enum EntityDetailsSectionLayout {
  /** Section spans the complete content width. */
  Full = "full",
  /** Section spans half of the content width. */
  Half = "half",
  /** Section spans one third of the content width. */
  Third = "third",
}

/** Semantic tones used by entity detail values. */
export enum EntityDetailTone {
  /** Default neutral tone. */
  Default = "default",
  /** Positive or successful value. */
  Positive = "positive",
  /** Warning value. */
  Warning = "warning",
  /** Negative or failed value. */
  Negative = "negative",
  /** Secondary muted value. */
  Muted = "muted",
}

/** Formatting modes supported by entity detail values. */
export enum EntityDetailValueFormat {
  /** Displays the value without transformation. */
  Default = "default",
  /** Formats enum values for human readability. */
  Enum = "enum",
}

/** A labeled value in an entity-details section. */
export interface EntityDetailItem {
  /** Value label. */
  label: string;
  /** Value content. */
  value: ReactNode;
  /** Semantic value tone. */
  tone?: EntityDetailTone;
  /** Uses a monospaced font for technical values. */
  mono?: boolean;
  /** Optional value formatting mode. */
  format?: EntityDetailValueFormat;
}

/** A highlighted metric in an entity-details section. */
export interface EntityDetailsMetric {
  /** Metric label. */
  label: string;
  /** Primary metric value. */
  value: ReactNode;
  /** Optional supporting detail. */
  detail?: ReactNode;
  /** Optional Material Symbols icon name. */
  icon?: string;
  /** Semantic metric tone. */
  tone?: EntityDetailTone;
  /** Uses a monospaced font for technical values. */
  mono?: boolean;
  /** Optional value formatting mode. */
  format?: EntityDetailValueFormat;
}

/** Empty state displayed when a details section has no data. */
export interface EntityDetailsEmptyState {
  /** Empty-state title. */
  title: string;
  /** Explanation shown below the title. */
  description: string;
  /** Optional Material Symbols icon name. */
  icon?: string;
}

/** Configures one section of the entity-details layout. */
export interface EntityDetailSection {
  /** Optional stable section identifier. */
  id?: string;
  /** Section title. */
  title: string;
  /** Optional section description. */
  description?: string;
  /** Optional Material Symbols icon name. */
  icon?: string;
  /** Labeled values displayed in the section. */
  items: Array<EntityDetailItem>;
  /** Highlighted metrics displayed in the section. */
  metrics?: Array<EntityDetailsMetric>;
  /** Empty state displayed when the section has no data. */
  emptyState?: EntityDetailsEmptyState;
  /** Section width within the details grid. */
  layout?: EntityDetailsSectionLayout;
}

/** Properties accepted by the entity-details component. */
export interface EntityDetailsProps {
  /** Sections rendered in order. */
  sections: Array<EntityDetailSection>;
}
