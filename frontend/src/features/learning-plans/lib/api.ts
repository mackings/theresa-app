import { apiFetch } from "@/lib/api";
import { createSession } from "@/lib/sessions";
import { LearningPlan, DurationUnit } from "@/features/learning-plans/types";

export function createLearningPlan(req: {
  title?: string;
  goal?: string;
  documentId?: string;
  durationValue: number;
  durationUnit: DurationUnit;
}): Promise<LearningPlan> {
  return apiFetch<LearningPlan>("/api/learning-plans", {
    method: "POST",
    body: JSON.stringify({
      title: req.title,
      goal: req.goal,
      document_id: req.documentId,
      duration_value: req.durationValue,
      duration_unit: req.durationUnit,
    }),
  });
}

export function listLearningPlans(): Promise<LearningPlan[]> {
  return apiFetch<LearningPlan[]>("/api/learning-plans");
}

export function getLearningPlan(id: string): Promise<LearningPlan> {
  return apiFetch<LearningPlan>(`/api/learning-plans/${id}`);
}

export function deleteLearningPlan(id: string): Promise<void> {
  return apiFetch(`/api/learning-plans/${id}`, { method: "DELETE" });
}

// startPlanStep is the one point this module reaches outside itself - it
// calls the existing, shared createSession (used by every other
// session-starting flow in the app already: the dashboard's quick-start
// cards, its ask-composer, its upload flow) rather than building a second
// session-creation path.
//
// mode defaults to "voice" - matching exactly how "Teach my course"'s
// existing document-upload flow already works (createSession("voice", ...)
// with no synthetic message at all, letting live_handler.go's opening-turn
// logic ground the first thing Theresa says). No pending-message hack is
// needed for voice: the backend snapshots this step's title/objectives onto
// the session (see SessionHandler.Create) and live_handler.go grounds the
// opening turn in them (live.LearningPlanStepPrompt) exactly the way it
// already grounds a document-upload voice session - real spoken teaching
// with the persona's normal pacing (stop every 2-3 boards, ask a real
// question, wait for a reply), not a one-shot text blitz through the whole
// topic.
//
// mode: "text" instead reuses the existing sessionStorage pending-message
// hand-off session/[id]/page.tsx already reads on mount - the exact same
// mechanism the dashboard's ask-composer already uses to open a text
// session with a first message.
export async function startPlanStep(
  plan: LearningPlan,
  step: { index: number; title: string },
  mode: "voice" | "text" = "voice"
) {
  const session = await createSession(mode, undefined, {
    learningPlanId: plan.id,
    learningPlanStepIndex: step.index,
  });

  if (mode === "text") {
    // Text mode still needs a first message to actually start teaching,
    // exactly like the dashboard's ask-composer - session creation alone
    // doesn't trigger anything in text mode the way it does for voice
    // (where live_handler.go's opening-turn logic fires the moment the WS
    // connection opens on the session page).
    sessionStorage.setItem(
      `theresa:pending-message:${session.id}`,
      JSON.stringify({ text: `Teach me: ${step.title}` })
    );
  }

  return session;
}
