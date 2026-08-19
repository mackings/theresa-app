"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Check, ClipboardCheck, Mic, MessageSquare, X } from "lucide-react";
import { Button, IconButton } from "@/components/ui/Button";
import { LearningPlan, LearningPlanStep } from "@/features/learning-plans/types";
import { startPlanStep } from "@/features/learning-plans/lib/api";
import { subjectPaletteFor } from "@/features/learning-plans/lib/subjectColor";
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
  const palette = subjectPaletteFor(plan.id);
  const startedCount = steps.filter((s) => s.session_id).length;
  // The first not-yet-started step gets a "start here" pulse - an
  // approximate progress marker, not a hard prerequisite gate (any step can
  // still be started out of order, same as before).
  const nextIndex = steps.findIndex((s) => !s.session_id);
  const progressPercent = steps.length > 0 ? (startedCount / steps.length) * 100 : 0;

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
      {/* The connecting path line - colored up to how far the plan's been
          started (an approximation, assuming roughly even step heights, not
          a pixel-measured value) and muted beyond that - gives the plan a
          real "path you walk through" feel, colored to match this plan's
          own subject identity (see PlanCard/ContinueLearning). */}
      <div
        aria-hidden
        className="absolute left-[23px] top-7 bottom-7 w-0.5 rounded-full"
        style={{
          background: `linear-gradient(to bottom, ${palette.ring} ${progressPercent}%, var(--color-border) ${progressPercent}%)`,
        }}
      />

      <div className="space-y-4">
        {steps.map((step) => {
          const started = !!step.session_id;
          const isNext = step.index === nextIndex;
          const isStarting = starting === step.index;
          return (
            <div key={step.index} className="relative flex gap-4">
              <div
                className={`z-10 flex h-12 w-12 shrink-0 items-center justify-center rounded-[var(--radius-full)] border-2 text-sm font-semibold transition-all ${
                  started
                    ? `${palette.solid} border-transparent text-white`
                    : isNext
                      ? `bg-[var(--color-surface-raised)] ${palette.text} animate-pulse`
                      : "border-[var(--color-border)] bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)]"
                }`}
                style={!started ? { borderColor: isNext ? palette.ring : undefined } : undefined}
              >
                {started ? <Check className="h-5 w-5" /> : step.index + 1}
              </div>

              <div className="flex-1 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-4 shadow-[var(--shadow-xs)] transition-shadow hover:shadow-[var(--shadow-sm)]">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className={`text-xs font-medium uppercase tracking-wide ${palette.text}`}>
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
                        <Check className={`mt-0.5 h-3 w-3 shrink-0 ${palette.text}`} />
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
