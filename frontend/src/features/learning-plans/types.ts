// TS mirrors of backend/internal/models/learningplan.go - deliberately a
// separate file from the shared @/types/board, since this module doesn't
// extend the board/session type surface, it has its own.

export type DurationUnit = "days" | "weeks" | "months";
export type LearningPlanStatus = "generating" | "ready" | "failed";

export interface LearningPlanStep {
  index: number;
  label: string;
  title: string;
  objectives?: string[];
  session_id?: string;
  // Set only when this step is specifically about pronouncing a non-English
  // language's sounds/words - a phonetic reference the voice persona reads
  // as an authoritative pronunciation guide. Not currently surfaced in the
  // UI (backend-consumed only), kept here just to mirror the Go type.
  pronunciation_notes?: string;
}

export interface LearningPlan {
  id: string;
  title: string;
  goal?: string;
  document_id?: string;
  duration_value: number;
  duration_unit: DurationUnit;
  status: LearningPlanStatus;
  error_message?: string;
  steps?: LearningPlanStep[];
  created_at: string;
  updated_at: string;
}
