"use client";

import { useRef, useState } from "react";
import { AlertCircle, FileCheck2, Loader2, Paperclip, RotateCcw, Send } from "lucide-react";
import { streamSessionMessage, ApiError } from "@/lib/api";
import { MessageBubble } from "@/components/chat/MessageBubble";
import { IconButton } from "@/components/ui/Button";
import { Pill } from "@/components/ui/Pill";
import { useDocumentUpload } from "@/lib/useDocumentUpload";
import { DocumentMeta, SessionEvent } from "@/types/board";

const SUGGESTIONS = [
  "Explain a concept: ",
  "Solve this problem: ",
  "Summarize my notes on ",
];

type Turn =
  | { role: "user"; seq: number; text: string }
  | { role: "assistant"; kind: "board"; seq: number; stepCount: number }
  | { role: "assistant"; kind: "chat"; seq: number; text: string };

function groupIntoTurns(events: SessionEvent[]): Turn[] {
  const sorted = [...events].sort((a, b) => a.seq - b.seq);
  const turns: Turn[] = [];

  for (const event of sorted) {
    if (event.type === "user_text") {
      turns.push({ role: "user", seq: event.seq, text: event.text ?? "" });
      continue;
    }

    // A real conversational check-in - its own bubble with real text, never
    // folded into the board step-count indicator below.
    if (event.type === "chat_message") {
      turns.push({ role: "assistant", kind: "chat", seq: event.seq, text: event.text ?? "" });
      continue;
    }

    const last = turns[turns.length - 1];
    if (last?.role === "assistant" && last.kind === "board") {
      last.stepCount += 1;
    } else {
      turns.push({ role: "assistant", kind: "board", seq: event.seq, stepCount: 1 });
    }
  }

  return turns;
}

export function ChatPanel({
  sessionId,
  events,
  documentId,
  onDocumentReady,
  onNewEvents,
  onSolvingChange,
}: {
  sessionId: string;
  events: SessionEvent[];
  documentId?: string;
  onDocumentReady: (doc: DocumentMeta) => void;
  onNewEvents: (events: SessionEvent[]) => void;
  onSolvingChange?: (solving: boolean) => void;
}) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { doc, error: uploadError, onFileSelected, reset: resetUpload } = useDocumentUpload(onDocumentReady);

  const turns = groupIntoTurns(events);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!text.trim() || sending) return;

    setError(null);
    setSending(true);
    onSolvingChange?.(true);
    const messageText = text;
    setText("");

    let firstBoardArrived = false;
    try {
      await streamSessionMessage(
        sessionId,
        { text: messageText, ...(documentId ? { document_id: documentId } : {}) },
        (event) => {
          onNewEvents([event]);
          // The send button only needs to reflect "is my message on its way
          // to Theresa" - once real content starts arriving, the board's own
          // pending/thinking indicator (still active until the full answer
          // finishes) is what should communicate ongoing generation, not a
          // send button stuck spinning for the whole multi-second answer.
          if (!firstBoardArrived && event.type === "board_update") {
            firstBoardArrived = true;
            setSending(false);
          }
        }
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
          ) : turn.kind === "chat" ? (
            <MessageBubble key={turn.seq} role="assistant" text={turn.text} />
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

        {doc && doc.status === "processing" && (
          <p className="mb-2 flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Processing {doc.filename}...
          </p>
        )}
        {doc && doc.status === "understood" && (
          <div className="mb-2 flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
            <FileCheck2 className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--color-accent)]" />
            <div>
              <p className="font-medium text-[var(--color-text-primary)]">{doc.filename}</p>
              <p className="mt-1">{doc.extracted_summary}</p>
            </div>
          </div>
        )}
        {uploadError && (
          <div className="mb-2 flex items-center justify-between gap-2 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-2 text-xs text-[var(--color-danger)]">
            <span className="flex items-center gap-1.5">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              {uploadError}
            </span>
            <IconButton
              variant="ghost"
              aria-label="Try again"
              onClick={resetUpload}
              className="h-6 w-6"
            >
              <RotateCcw className="h-3.5 w-3.5" />
            </IconButton>
          </div>
        )}

        <div className="flex items-center gap-2 rounded-[var(--radius-full)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] py-1 pl-2 pr-1.5 shadow-[var(--shadow-xs)]">
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,.jpg,.jpeg,.png"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) onFileSelected(file);
              e.target.value = "";
            }}
          />
          <IconButton
            type="button"
            variant="ghost"
            aria-label="Upload a PDF or photo of a page"
            onClick={() => fileInputRef.current?.click()}
            className="h-8 w-8 shrink-0 rounded-[var(--radius-full)]"
          >
            <Paperclip className="h-4 w-4" />
          </IconButton>
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
