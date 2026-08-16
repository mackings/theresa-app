"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { AlertCircle, Calendar, FileText, NotebookPen, Sparkles, Target } from "lucide-react";
import { Card, CardHeader } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { UploadDropzone } from "@/components/chat/UploadDropzone";
import { DocumentMeta } from "@/types/board";
import { createLearningPlan } from "@/features/learning-plans/lib/api";
import { DurationUnit } from "@/features/learning-plans/types";

const DURATION_UNITS: DurationUnit[] = ["days", "weeks", "months"];

function SegmentedToggle<T extends string>({
  options,
  value,
  onChange,
  labels,
  icons,
}: {
  options: T[];
  value: T;
  onChange: (v: T) => void;
  labels?: Partial<Record<T, string>>;
  icons?: Partial<Record<T, React.ReactNode>>;
}) {
  return (
    <div
      role="group"
      className="flex items-center gap-0.5 rounded-[var(--radius-full)] border border-[var(--color-border)] bg-[var(--color-surface)] p-0.5"
    >
      {options.map((opt) => (
        <button
          key={opt}
          type="button"
          onClick={() => onChange(opt)}
          aria-pressed={value === opt}
          className={`flex flex-1 items-center justify-center gap-1.5 rounded-[var(--radius-full)] px-3 py-1.5 text-xs font-medium capitalize transition-colors ${
            value === opt
              ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)]"
              : "text-[var(--color-text-secondary)]"
          }`}
        >
          {icons?.[opt]}
          {labels?.[opt] ?? opt}
        </button>
      ))}
    </div>
  );
}

export function PlanForm() {
  const router = useRouter();
  const [source, setSource] = useState<"document" | "goal">("goal");
  const [document, setDocument] = useState<DocumentMeta | null>(null);
  const [goal, setGoal] = useState("");
  const [durationValue, setDurationValue] = useState(2);
  const [durationUnit, setDurationUnit] = useState<DurationUnit>("weeks");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit =
    !creating && durationValue >= 1 && durationValue <= 52 &&
    (source === "goal" ? goal.trim().length > 0 : document !== null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;
    setCreating(true);
    setError(null);
    try {
      const plan = await createLearningPlan({
        goal: source === "goal" ? goal.trim() : undefined,
        documentId: source === "document" ? document!.id : undefined,
        durationValue,
        durationUnit,
      });
      router.push(`/learning-plans/${plan.id}`);
    } catch {
      setError("We couldn't create your plan - please try again.");
      setCreating(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <Card className="overflow-hidden">
        <CardHeader icon={<Target className="h-3.5 w-3.5" />} title="What do you want to learn?" />
        <div className="p-5 pt-3">
          <SegmentedToggle
            options={["goal", "document"] as const}
            value={source}
            onChange={setSource}
            labels={{ goal: "State a goal", document: "Use a document" }}
            icons={{
              goal: <Target className="h-3.5 w-3.5" />,
              document: <FileText className="h-3.5 w-3.5" />,
            }}
          />

          <div className="mt-4">
            {source === "goal" ? (
              <textarea
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
                placeholder="e.g. I want to learn Python programming from scratch"
                rows={3}
                className="w-full resize-none rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-3.5 py-3 text-sm text-[var(--color-text-primary)] outline-none transition-colors focus:border-[var(--color-accent)]"
              />
            ) : (
              <UploadDropzone onDocumentReady={setDocument} />
            )}
          </div>
        </div>
      </Card>

      <Card className="overflow-hidden">
        <CardHeader icon={<Calendar className="h-3.5 w-3.5" />} title="How long should this plan take?" />
        <div className="flex items-center gap-3 p-5 pt-3">
          <input
            type="number"
            min={1}
            max={52}
            value={durationValue}
            onChange={(e) => setDurationValue(Number(e.target.value))}
            className="w-20 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-3.5 py-2.5 text-center text-sm font-medium text-[var(--color-text-primary)] outline-none transition-colors focus:border-[var(--color-accent)]"
          />
          <SegmentedToggle options={DURATION_UNITS} value={durationUnit} onChange={setDurationUnit} />
        </div>
      </Card>

      {error && (
        <p className="flex items-center gap-1.5 text-sm text-[var(--color-danger)]">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
        </p>
      )}

      <Button
        type="submit"
        variant="primary"
        icon={creating ? undefined : <Sparkles className="h-4 w-4" />}
        disabled={!canSubmit}
        className="w-full py-3 text-sm"
      >
        {creating ? (
          <span className="flex items-center gap-2">
            <NotebookPen className="h-4 w-4 animate-pulse" />
            Creating your plan…
          </span>
        ) : (
          "Create my lesson plan"
        )}
      </Button>
    </form>
  );
}
