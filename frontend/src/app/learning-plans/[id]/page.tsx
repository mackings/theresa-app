"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { AlertCircle, ArrowLeft, Loader2, NotebookPen } from "lucide-react";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { IconButton } from "@/components/ui/Button";
import { PlanStepList } from "@/features/learning-plans/components/PlanStepList";
import { getLearningPlan } from "@/features/learning-plans/lib/api";
import { LearningPlan } from "@/features/learning-plans/types";
import { subjectPaletteFor } from "@/features/learning-plans/lib/subjectColor";

// Same poll-while-processing idiom useDocumentUpload.ts already uses for
// document understanding - matches the backend's own generatePlan shape
// (status "generating" -> "ready"|"failed").
const POLL_INTERVAL_MS = 2000;
const MAX_POLL_DURATION_MS = 150_000;

export default function LearningPlanDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [plan, setPlan] = useState<LearningPlan | null>(null);
  const [timedOut, setTimedOut] = useState(false);
  const cancelledRef = useRef(false);

  useEffect(() => {
    cancelledRef.current = false;
    const deadline = Date.now() + MAX_POLL_DURATION_MS;

    async function poll() {
      while (!cancelledRef.current) {
        let updated: LearningPlan;
        try {
          updated = await getLearningPlan(params.id);
        } catch {
          router.replace("/learning-plans");
          return;
        }
        if (cancelledRef.current) return;
        setPlan(updated);

        if (updated.status !== "generating") return;
        if (Date.now() > deadline) {
          setTimedOut(true);
          return;
        }
        await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
      }
    }

    poll();
    return () => {
      cancelledRef.current = true;
    };
  }, [params.id, router]);

  if (!plan) return null;

  const steps = plan.steps ?? [];
  const startedCount = steps.filter((s) => s.session_id).length;
  const palette = subjectPaletteFor(plan.id);

  return (
    <AppShell>
      <div className="mx-auto max-w-2xl space-y-6 px-6 py-10">
        <div className="flex items-center gap-3">
          <IconButton
            variant="ghost"
            aria-label="Back to learning plans"
            onClick={() => router.push("/learning-plans")}
          >
            <ArrowLeft className="h-4 w-4" />
          </IconButton>
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
              <span className={`flex h-8 w-8 items-center justify-center rounded-[var(--radius-md)] ${palette.bg} ${palette.text}`}>
                <NotebookPen className="h-4 w-4" />
              </span>
              {plan.title}
            </h1>
            <p className="mt-0.5 text-sm text-[var(--color-text-secondary)]">
              {plan.duration_value} {plan.duration_unit}
              {steps.length > 0 && ` · ${startedCount} of ${steps.length} started`}
            </p>
          </div>
        </div>

        {plan.status === "generating" && !timedOut && (
          <Card className="flex flex-col items-center gap-3 px-6 py-14 text-center">
            <Loader2 className="h-6 w-6 animate-spin text-[var(--color-accent)]" />
            <p className="text-sm text-[var(--color-text-secondary)]">
              Theresa is putting together your plan…
            </p>
          </Card>
        )}

        {plan.status === "generating" && timedOut && (
          <Card className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <AlertCircle className="h-5 w-5 text-[var(--color-danger)]" />
            <p className="text-sm text-[var(--color-text-secondary)]">
              This is taking longer than expected. Reopen this page to check again.
            </p>
          </Card>
        )}

        {plan.status === "failed" && (
          <Card className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <AlertCircle className="h-5 w-5 text-[var(--color-danger)]" />
            <p className="text-sm text-[var(--color-text-secondary)]">
              {plan.error_message ?? "We couldn't generate this plan."}
            </p>
          </Card>
        )}

        {plan.status === "ready" && <PlanStepList plan={plan} />}
      </div>
    </AppShell>
  );
}
