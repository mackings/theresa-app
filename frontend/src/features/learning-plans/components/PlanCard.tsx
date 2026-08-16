import { Loader2, NotebookPen, AlertCircle } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { LearningPlan } from "@/features/learning-plans/types";

const STATUS_PILL: Record<LearningPlan["status"], { label: string; className: string }> = {
  generating: {
    label: "Generating",
    className: "bg-[var(--color-accent)]/10 text-[var(--color-accent)]",
  },
  ready: {
    label: "Ready",
    className: "bg-[var(--color-accent)]/10 text-[var(--color-accent)]",
  },
  failed: {
    label: "Failed",
    className: "bg-[var(--color-danger)]/10 text-[var(--color-danger)]",
  },
};

export function PlanCard({
  plan,
  onClick,
}: {
  plan: LearningPlan;
  onClick: () => void;
}) {
  const pill = STATUS_PILL[plan.status];
  const steps = plan.steps ?? [];
  const startedCount = steps.filter((s) => s.session_id).length;
  const progress = steps.length > 0 ? startedCount / steps.length : 0;

  return (
    <Card
      onClick={onClick}
      className="cursor-pointer overflow-hidden transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
    >
      <div className="flex items-center justify-center bg-[var(--color-surface)] p-6">
        <div className="flex h-12 w-12 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-accent)]/10 text-[var(--color-accent)]">
          <NotebookPen className="h-5 w-5" />
        </div>
      </div>
      <div className="p-4">
        <span
          className={`flex w-fit items-center gap-1 rounded-[var(--radius-full)] px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide ${pill.className}`}
        >
          {plan.status === "generating" && <Loader2 className="h-3 w-3 animate-spin" />}
          {plan.status === "failed" && <AlertCircle className="h-3 w-3" />}
          {pill.label}
        </span>
        <p className="mt-2 truncate text-sm font-semibold text-[var(--color-text-primary)]">
          {plan.title}
        </p>
        <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
          {plan.duration_value} {plan.duration_unit}
          {steps.length > 0 ? ` · ${steps.length} steps` : ""}
        </p>

        {steps.length > 0 && (
          <div className="mt-3">
            <div className="h-1.5 w-full overflow-hidden rounded-[var(--radius-full)] bg-[var(--color-surface)]">
              <div
                className="h-full rounded-[var(--radius-full)] bg-[var(--color-accent)] transition-all"
                style={{ width: `${Math.max(progress * 100, startedCount > 0 ? 6 : 0)}%` }}
              />
            </div>
            <p className="mt-1.5 text-[11px] text-[var(--color-text-secondary)]">
              {startedCount} of {steps.length} started
            </p>
          </div>
        )}
      </div>
    </Card>
  );
}
