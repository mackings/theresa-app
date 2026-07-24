import { Sparkles } from "lucide-react";

export function MessageBubble({
  role,
  text,
  stepCount,
}: {
  role: "user" | "assistant";
  text?: string;
  stepCount?: number;
}) {
  if (role === "user") {
    return (
      <div className="ml-auto max-w-[85%] rounded-[var(--radius-lg)] bg-[var(--color-accent)] px-3.5 py-2 text-sm text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)]">
        {text}
      </div>
    );
  }

  return (
    <div className="mr-auto flex max-w-[85%] items-center gap-2">
      <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-[var(--color-accent-foreground)]">
        <Sparkles className="h-3.5 w-3.5" />
      </div>
      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] px-3.5 py-2 text-sm text-[var(--color-text-secondary)]">
        Explained on the board — {stepCount} step{stepCount === 1 ? "" : "s"}
      </div>
    </div>
  );
}
