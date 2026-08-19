import { Loader2, NotebookPen, AlertCircle } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { ProgressRing } from "@/components/ui/ProgressRing";
import { LearningPlan } from "@/features/learning-plans/types";
import { subjectPaletteFor } from "@/features/learning-plans/lib/subjectColor";

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
  const palette = subjectPaletteFor(plan.id);

  return (
    <Card
      onClick={onClick}
      className="cursor-pointer overflow-hidden transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
    >
      <div className={`flex items-center justify-center p-6 ${palette.bg}`}>
        <ProgressRing progress={progress} color={palette.ring} size={56} strokeWidth={4}>
          <div className={`flex h-9 w-9 items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-surface-raised)] ${palette.text}`}>
            <NotebookPen className="h-4 w-4" />
          </div>
        </ProgressRing>
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
          {steps.length > 0 ? ` · ${startedCount} of ${steps.length} started` : ""}
        </p>
      </div>
    </Card>
  );
}
