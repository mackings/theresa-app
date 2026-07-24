"use client";

import { useState } from "react";
import { AlertCircle, Loader2, Send } from "lucide-react";
import { streamSessionMessage, ApiError } from "@/lib/api";
import { MessageBubble } from "@/components/chat/MessageBubble";
import { IconButton } from "@/components/ui/Button";
import { Pill } from "@/components/ui/Pill";
import { SessionEvent } from "@/types/board";

const SUGGESTIONS = [
  "Explain a concept: ",
  "Solve this problem: ",
  "Summarize my notes on ",
];

type Turn =
  | { role: "user"; seq: number; text: string }
  | { role: "assistant"; seq: number; stepCount: number };

function groupIntoTurns(events: SessionEvent[]): Turn[] {
  const sorted = [...events].sort((a, b) => a.seq - b.seq);
  const turns: Turn[] = [];

  for (const event of sorted) {
    if (event.type === "user_text") {
      turns.push({ role: "user", seq: event.seq, text: event.text ?? "" });
      continue;
    }

    const last = turns[turns.length - 1];
    if (last?.role === "assistant") {
      last.stepCount += 1;
    } else {
      turns.push({ role: "assistant", seq: event.seq, stepCount: 1 });
    }
  }

  return turns;
}

export function ChatPanel({
  sessionId,
  events,
  documentId,
  onNewEvents,
  onSolvingChange,
}: {
  sessionId: string;
  events: SessionEvent[];
  documentId?: string;
  onNewEvents: (events: SessionEvent[]) => void;
  onSolvingChange?: (solving: boolean) => void;
}) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const turns = groupIntoTurns(events);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!text.trim() || sending) return;

    setError(null);
    setSending(true);
    onSolvingChange?.(true);
    const messageText = text;
    setText("");

    try {
      await streamSessionMessage(
        sessionId,
        { text: messageText, ...(documentId ? { document_id: documentId } : {}) },
        (event) => onNewEvents([event])
      );
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "something went wrong");
    } finally {
      setSending(false);
      onSolvingChange?.(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 space-y-2 overflow-y-auto p-4">
        {turns.map((turn) =>
          turn.role === "user" ? (
            <MessageBubble key={turn.seq} role="user" text={turn.text} />
          ) : (
            <MessageBubble key={turn.seq} role="assistant" stepCount={turn.stepCount} />
          )
        )}
      </div>

      <form onSubmit={onSubmit} className="border-t border-[var(--color-border)] p-3">
        {turns.length === 0 && (
          <div className="mb-3 flex flex-wrap gap-2">
            {SUGGESTIONS.map((suggestion) => (
              <button key={suggestion} type="button" onClick={() => setText(suggestion)}>
                <Pill className="cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)]">
                  {suggestion.replace(/:\s*$/, "")}
                </Pill>
              </button>
            ))}
          </div>
        )}
        {error && (
          <p className="mb-2 flex items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-1.5 text-xs text-[var(--color-danger)]">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
            {error}
          </p>
        )}
        <div className="flex items-center gap-2 rounded-[var(--radius-full)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] py-1 pl-4 pr-1.5 shadow-[var(--shadow-xs)]">
          <input
            type="text"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Paste a problem or ask a question..."
            className="flex-1 bg-transparent py-1.5 text-sm text-[var(--color-text-primary)] outline-none"
          />
          <IconButton
            type="submit"
            variant="primary"
            disabled={sending || !text.trim()}
            aria-label="Send message"
            className="h-8 w-8 rounded-[var(--radius-full)]"
          >
            {sending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </IconButton>
        </div>
      </form>
    </div>
  );
}
