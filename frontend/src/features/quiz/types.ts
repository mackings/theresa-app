// TS mirror of backend/internal/httpapi/quiz_handlers.go's quizView/
// quizQuestionView - deliberately matches the API's own redaction: grading
// fields are only ever present once attempted (backend omits them entirely
// pre-submission, not just "sets them to null").
export interface QuizQuestion {
  prompt: string;
  options: string[];
  correct_index?: number;
  selected_index?: number;
  correct?: boolean;
}

export interface Quiz {
  id: string;
  session_id: string;
  status: "generating" | "ready" | "failed";
  attempted: boolean;
  score?: number;
  total_count?: number;
  questions?: QuizQuestion[];
}
