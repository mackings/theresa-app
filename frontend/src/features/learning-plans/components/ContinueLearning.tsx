"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, Mic, MessageSquare } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { relativeTime } from "@/lib/time";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { TutorSession } from "@/types/board";
import { LearningPlan, LearningPlanStep } from "@/features/learning-plans/types";
import { subjectPaletteFor } from "@/features/learning-plans/lib/subjectColor";

const MAX_RESUME_ITEMS = 4;

// Surfaces "where a student/org last left off" so a class can be resumed in
// one click instead of hunting through the plan/step list - built entirely
// from the existing session list (already sorted by updated_at) filtered to
// sessions that came from a learning-plan step, deduped to the single most
// recent session per plan. Each card's color matches that plan's own
// subject color from PlanCard below, so the same course reads as the same
// visual identity in both places.
export function ContinueLearning({ plans }: { plans: LearningPlan[] }) {
  const router = useRouter();
  const [sessions, setSessions] = useState<TutorSession[] | null>(null);

  useEffect(() => {
    apiFetch<TutorSession[]>("/api/sessions")
      .then(setSessions)
      .catch(() => setSessions([]));
  }, []);

  if (!sessions) return null;

  const planById = new Map(plans.map((p) => [p.id, p]));
  const seenPlans = new Set<string>();
  const items: { session: TutorSession; plan: LearningPlan; step?: LearningPlanStep }[] = [];

  for (const session of sessions) {
    if (!session.learning_plan_id) continue;
    const plan = planById.get(session.learning_plan_id);
    if (!plan || seenPlans.has(plan.id)) continue;
    seenPlans.add(plan.id);
    const step = plan.steps?.find((s) => s.index === session.learning_plan_step_index);
    items.push({ session, plan, step });
    if (items.length >= MAX_RESUME_ITEMS) break;
  }

  if (items.length === 0) return null;

  return (
    <div>
      <p className="mb-3 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
        Continue learning
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {items.map(({ session, plan, step }) => {
          const isVoice = session.mode === "voice";
          const ModeIcon = isVoice ? Mic : MessageSquare;
          const palette = subjectPaletteFor(plan.id);
          return (
            <Card
              key={session.id}
              onClick={() => router.push(`/session/${session.id}`)}
              className="group cursor-pointer overflow-hidden transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
            >
              <div className={`h-1.5 w-full ${palette.solid}`} />
              <div className="flex items-center gap-3 p-4">
                <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius-full)] ${palette.bg} ${palette.text}`}>
                  <ModeIcon className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className={`truncate text-xs font-medium ${palette.text}`}>{plan.title}</p>
                  <p className="mt-0.5 truncate text-sm font-semibold text-[var(--color-text-primary)]">
                    {step ? `${step.label}: ${step.title}` : session.title}
                  </p>
                  <p className="mt-0.5 text-xs text-[var(--color-text-secondary)]">
                    Last studied {relativeTime(session.updated_at)}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  className="shrink-0 !px-2.5 !py-1.5 text-xs opacity-0 transition-opacity group-hover:opacity-100"
                  icon={<ArrowRight className="h-3.5 w-3.5" />}
                >
                  Resume
                </Button>
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
