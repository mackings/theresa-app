export function MessageBubble({
  role,
  text,
}: {
  role: "user" | "assistant";
  text: string;
}) {
  if (role === "user") {
    return (
      <div className="ml-auto max-w-[85%] rounded-[var(--radius-lg)] bg-[var(--color-accent)] px-3.5 py-2 text-sm text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)]">
        {text}
      </div>
    );
  }

  return (
    <div className="mr-auto max-w-[85%] rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] px-3.5 py-2 text-sm text-[var(--color-text-primary)]">
      {text}
    </div>
  );
}
