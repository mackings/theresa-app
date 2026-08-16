"use client";

import { useEffect, useRef, useState } from "react";
import { AlertCircle, Check, Loader2, X } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { getOrCreateQuiz, getQuiz, submitQuiz } from "@/features/quiz/lib/api";
import { Quiz } from "@/features/quiz/types";

// Same poll-while-processing idiom already established in
// app/learning-plans/[id]/page.tsx - a shorter MAX_POLL_DURATION_MS here
// since one quiz is far smaller than a whole plan.
const POLL_INTERVAL_MS = 2000;
const MAX_POLL_DURATION_MS = 60_000;

export function QuizPanel({ sessionId }: { sessionId: string }) {
  const [quiz, setQuiz] = useState<Quiz | null>(null);
  const [timedOut, setTimedOut] = useState(false);
  const [answers, setAnswers] = useState<Record<number, number>>({});
  const [submitting, setSubmitting] = useState(false);
  const cancelledRef = useRef(false);

  useEffect(() => {
    cancelledRef.current = false;
    const deadline = Date.now() + MAX_POLL_DURATION_MS;

    async function start() {
      let current: Quiz;
      try {
        current = await getOrCreateQuiz(sessionId);
      } catch {
        return;
      }
      if (cancelledRef.current) return;
      setQuiz(current);

      while (!cancelledRef.current && current.status === "generating") {
        if (Date.now() > deadline) {
          setTimedOut(true);
          return;
        }
        await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
        if (cancelledRef.current) return;
        try {
          current = await getQuiz(sessionId);
        } catch {
          return;
        }
        if (cancelledRef.current) return;
        setQuiz(current);
      }
    }

    start();
    return () => {
      cancelledRef.current = true;
    };
  }, [sessionId]);

  async function handleSubmit() {
    if (!quiz?.questions || submitting) return;
    setSubmitting(true);
    try {
      const orderedAnswers = quiz.questions.map((_, i) => answers[i]);
      const result = await submitQuiz(sessionId, orderedAnswers);
      setQuiz(result);
    } finally {
      setSubmitting(false);
    }
  }

  if (!quiz || (quiz.status === "generating" && !timedOut)) {
    return (
      <Card className="flex flex-col items-center gap-3 px-6 py-14 text-center">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--color-accent)]" />
        <p className="text-sm text-[var(--color-text-secondary)]">
          Theresa is putting together a quiz for this lesson…
        </p>
      </Card>
    );
  }

  if (quiz.status === "generating" && timedOut) {
    return (
      <Card className="flex flex-col items-center gap-2 px-6 py-14 text-center">
        <AlertCircle className="h-5 w-5 text-[var(--color-danger)]" />
        <p className="text-sm text-[var(--color-text-secondary)]">
          This is taking longer than expected. Reopen this to check again.
        </p>
      </Card>
    );
  }

  if (quiz.status === "failed") {
    return (
      <Card className="flex flex-col items-center gap-2 px-6 py-14 text-center">
        <AlertCircle className="h-5 w-5 text-[var(--color-danger)]" />
        <p className="text-sm text-[var(--color-text-secondary)]">
          We couldn&apos;t generate a quiz for this lesson.
        </p>
      </Card>
    );
  }

  const questions = quiz.questions ?? [];

  if (questions.length === 0) {
    return (
      <Card className="flex flex-col items-center gap-2 px-6 py-14 text-center">
        <p className="text-sm text-[var(--color-text-secondary)]">
          Not enough was covered in this lesson yet for a meaningful quiz.
        </p>
      </Card>
    );
  }

  if (quiz.attempted) {
    return (
      <div className="space-y-4">
        <Card className="px-5 py-4 text-center">
          <p className="text-lg font-semibold text-[var(--color-text-primary)]">
            {quiz.score} / {quiz.total_count} correct
          </p>
        </Card>
        {questions.map((q, i) => (
          <Card key={i} className="p-4">
            <p className="text-sm font-semibold text-[var(--color-text-primary)]">{q.prompt}</p>
            <div className="mt-3 space-y-1.5">
              {q.options.map((option, oi) => {
                const isCorrect = oi === q.correct_index;
                const isSelected = oi === q.selected_index;
                return (
                  <div
                    key={oi}
                    className={`flex items-center gap-2 rounded-[var(--radius-md)] border px-3 py-2 text-sm ${
                      isCorrect
                        ? "border-[var(--color-accent)] bg-[var(--color-accent)]/10 text-[var(--color-text-primary)]"
                        : isSelected
                          ? "border-[var(--color-danger)] bg-[var(--color-danger)]/10 text-[var(--color-text-primary)]"
                          : "border-[var(--color-border-subtle)] text-[var(--color-text-secondary)]"
                    }`}
                  >
                    {isCorrect ? (
                      <Check className="h-3.5 w-3.5 shrink-0 text-[var(--color-accent)]" />
                    ) : isSelected ? (
                      <X className="h-3.5 w-3.5 shrink-0 text-[var(--color-danger)]" />
                    ) : (
                      <span className="h-3.5 w-3.5 shrink-0" />
                    )}
                    {option}
                  </div>
                );
              })}
            </div>
          </Card>
        ))}
      </div>
    );
  }

  const allAnswered = questions.every((_, i) => answers[i] !== undefined);

  return (
    <div className="space-y-4">
      {questions.map((q, i) => (
        <Card key={i} className="p-4">
          <p className="text-sm font-semibold text-[var(--color-text-primary)]">{q.prompt}</p>
          <div className="mt-3 space-y-1.5">
            {q.options.map((option, oi) => (
              <button
                key={oi}
                type="button"
                onClick={() => setAnswers((prev) => ({ ...prev, [i]: oi }))}
                className={`flex w-full items-center gap-2 rounded-[var(--radius-md)] border px-3 py-2 text-left text-sm transition-colors ${
                  answers[i] === oi
                    ? "border-[var(--color-accent)] bg-[var(--color-accent)]/10 text-[var(--color-text-primary)]"
                    : "border-[var(--color-border-subtle)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)]"
                }`}
              >
                {option}
              </button>
            ))}
          </div>
        </Card>
      ))}
      <Button variant="primary" disabled={!allAnswered || submitting} onClick={handleSubmit} className="w-full">
        {submitting ? "Submitting…" : "Submit"}
      </Button>
    </div>
  );
}
