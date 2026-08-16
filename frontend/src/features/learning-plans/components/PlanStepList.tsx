"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Check, ClipboardCheck, Mic, MessageSquare, X } from "lucide-react";
import { Button, IconButton } from "@/components/ui/Button";
import { LearningPlan, LearningPlanStep } from "@/features/learning-plans/types";
import { startPlanStep } from "@/features/learning-plans/lib/api";
import { QuizPanel } from "@/features/quiz/components/QuizPanel";

export function PlanStepList({ plan }: { plan: LearningPlan }) {
  const router = useRouter();
  const [starting, setStarting] = useState<number | null>(null);
  // The quiz is a learning-plan-only feature (not available for plain
  // voice/text sessions - enforced server-side too, see QuizHandler), so
  // its entire UI - trigger and taking it - lives here on the plan page
  // rather than on the general session page.
  const [quizSessionId, setQuizSessionId] = useState<string | null>(null);
  const steps = plan.steps ?? [];

  async function handleStart(step: LearningPlanStep, mode: "voice" | "text") {
    if (starting !== null) return;
    if (step.session_id) {
      router.push(`/session/${step.session_id}`);
      return;
    }
    setStarting(step.index);
    try {
      const session = await startPlanStep(plan, step, mode);
      router.push(`/session/${session.id}`);
    } finally {
      setStarting(null);
    }
  }

  return (
    <div className="relative">
      {/* The connecting timeline line - positioned behind the numbered
          circles, spanning from the first to the last step. Purely
          decorative, gives the plan a "path you walk through" feel instead
          of a flat list of unrelated cards. */}
      <div
        aria-hidden
        className="absolute left-[19px] top-6 bottom-6 w-px bg-[var(--color-border)]"
      />

      <div className="space-y-4">
        {steps.map((step) => {
          const started = !!step.session_id;
          const isStarting = starting === step.index;
          return (
            <div key={step.index} className="relative flex gap-4">
              <div
                className={`z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius-full)] border text-sm font-semibold ${
                  started
                    ? "border-[var(--color-accent)] bg-[var(--color-accent)] text-[var(--color-accent-foreground)]"
                    : "border-[var(--color-border)] bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)]"
                }`}
              >
                {started ? <Check className="h-4 w-4" /> : step.index + 1}
              </div>

              <div className="flex-1 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-4 shadow-[var(--shadow-xs)] transition-shadow hover:shadow-[var(--shadow-sm)]">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wide text-[var(--color-accent)]">
                      {step.label}
                    </p>
                    <p className="mt-0.5 text-sm font-semibold text-[var(--color-text-primary)]">
                      {step.title}
                    </p>
                  </div>

                  {started ? (
                    <div className="flex shrink-0 items-center gap-1.5">
                      <IconButton
                        variant="ghost"
                        aria-label={`Test yourself on "${step.title}"`}
                        onClick={() => setQuizSessionId(step.session_id!)}
                        className="h-9 w-9 border border-[var(--color-border)]"
                      >
                        <ClipboardCheck className="h-4 w-4" />
                      </IconButton>
                      <Button
                        variant="secondary"
                        onClick={() => handleStart(step, "voice")}
                      >
                        Continue
                      </Button>
                    </div>
                  ) : (
                    <div className="flex shrink-0 items-center gap-1.5">
                      <IconButton
                        variant="ghost"
                        aria-label={`Start "${step.title}" as text`}
                        disabled={starting !== null}
                        onClick={() => handleStart(step, "text")}
                        className="h-9 w-9 border border-[var(--color-border)]"
                      >
                        <MessageSquare className="h-4 w-4" />
                      </IconButton>
                      <Button
                        variant="primary"
                        icon={<Mic className="h-3.5 w-3.5" />}
                        disabled={starting !== null}
                        onClick={() => handleStart(step, "voice")}
                      >
                        {isStarting ? "Starting…" : "Start"}
                      </Button>
                    </div>
                  )}
                </div>

                {step.objectives && step.objectives.length > 0 && (
                  <ul className="mt-3 space-y-1 border-t border-[var(--color-border-subtle)] pt-3">
                    {step.objectives.map((o, i) => (
                      <li
                        key={i}
                        className="flex items-start gap-1.5 text-xs text-[var(--color-text-secondary)]"
                      >
                        <Check className="mt-0.5 h-3 w-3 shrink-0 text-[var(--color-accent)]" />
                        {o}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {quizSessionId && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => setQuizSessionId(null)}
        >
          <div
            role="dialog"
            aria-modal="true"
            className="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-5 shadow-[var(--shadow-lg)]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center justify-between">
              <p className="flex items-center gap-1.5 text-sm font-semibold text-[var(--color-text-primary)]">
                <ClipboardCheck className="h-4 w-4 text-[var(--color-accent)]" />
                Test yourself
              </p>
              <IconButton
                variant="ghost"
                aria-label="Close quiz"
                onClick={() => setQuizSessionId(null)}
              >
                <X className="h-4 w-4" />
              </IconButton>
            </div>
            <QuizPanel sessionId={quizSessionId} />
          </div>
        </div>
      )}
    </div>
  );
}
