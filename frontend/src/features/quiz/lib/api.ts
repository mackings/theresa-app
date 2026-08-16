import { apiFetch } from "@/lib/api";
import { Quiz } from "@/features/quiz/types";

// Quizzes only exist for sessions started from a learning-plan step - the
// backend rejects these calls (403) for any other session, so these are
// only ever called from within the learning-plans module.
export function getOrCreateQuiz(sessionId: string): Promise<Quiz> {
  return apiFetch<Quiz>(`/api/sessions/${sessionId}/quiz`, { method: "POST" });
}

export function getQuiz(sessionId: string): Promise<Quiz> {
  return apiFetch<Quiz>(`/api/sessions/${sessionId}/quiz`);
}

export function submitQuiz(sessionId: string, answers: number[]): Promise<Quiz> {
  return apiFetch<Quiz>(`/api/sessions/${sessionId}/quiz/submit`, {
    method: "POST",
    body: JSON.stringify({ answers }),
  });
}
