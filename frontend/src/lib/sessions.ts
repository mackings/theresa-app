import { apiFetch } from "@/lib/api";
import { TutorSession } from "@/types/board";

export async function createSession(
  mode: "text" | "voice",
  documentIds?: string[],
  // Optional linkage to a specific learning-plan step (see
  // features/learning-plans/lib/api.ts's startPlanStep) - omitted for every
  // other caller, which is the vast majority of session creation.
  learningPlan?: { learningPlanId: string; learningPlanStepIndex: number }
): Promise<TutorSession> {
  return apiFetch<TutorSession>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({
      mode,
      ...(documentIds ? { document_ids: documentIds } : {}),
      ...(learningPlan
        ? {
            learning_plan_id: learningPlan.learningPlanId,
            learning_plan_step_index: learningPlan.learningPlanStepIndex,
          }
        : {}),
    }),
  });
}
