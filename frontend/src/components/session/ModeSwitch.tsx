import { MessageSquare, Mic } from "lucide-react";

export function ModeSwitch({
  mode,
  onChange,
  disabled,
}: {
  mode: "text" | "voice";
  onChange: (mode: "text" | "voice") => void;
  disabled?: boolean;
}) {
  return (
    <div
      role="group"
      aria-label="Session mode"
      className="flex items-center gap-0.5 rounded-[var(--radius-full)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-0.5"
    >
      <button
        type="button"
        onClick={() => onChange("text")}
        disabled={disabled}
        aria-pressed={mode === "text"}
        className={`flex items-center gap-1.5 rounded-[var(--radius-full)] px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-60 ${
          mode === "text"
            ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)]"
            : "text-[var(--color-text-secondary)]"
        }`}
      >
        <MessageSquare className="h-3.5 w-3.5" />
        Text
      </button>
      <button
        type="button"
        onClick={() => onChange("voice")}
        disabled={disabled}
        aria-pressed={mode === "voice"}
        className={`flex items-center gap-1.5 rounded-[var(--radius-full)] px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-60 ${
          mode === "voice"
            ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)]"
            : "text-[var(--color-text-secondary)]"
        }`}
      >
        <Mic className="h-3.5 w-3.5" />
        Voice
      </button>
    </div>
  );
}
