"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { AlertCircle, Loader2 } from "lucide-react";
import { apiFetch, ApiError } from "@/lib/api";
import { createSession } from "@/lib/sessions";
import { Board } from "@/components/board/Board";
import { VoiceControls } from "@/components/voice/VoiceControls";
import { BoardAudioSync } from "@/lib/board/audioSync";
import { TutorSession, SessionEvent, BoardContentBlock } from "@/types/board";

// Starts a real, ephemeral demo account + voice session (see backend's
// DemoHandler) and renders the exact same Board + VoiceControls a signed-up
// user gets. Shared by the dedicated /try page and the landing page's
// inline "Start Learning now" flow, which mounts this in place instead of
// navigating away - both just need a parent that gives it real height
// (a flex column with a set height works, same as /try's own wrapper).
export function DemoExperience() {
  const [session, setSession] = useState<TutorSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [events, setEvents] = useState<SessionEvent[]>([]);
  const nextSeqRef = useRef(0);
  const [audioSync] = useState(() => new BoardAudioSync());
  const startedRef = useRef(false);

  useEffect(() => {
    // React Strict Mode double-invokes effects in dev - without this guard,
    // that would mint two ephemeral demo accounts (and burn two of the
    // per-IP rate limit's three slots) on a single mount.
    if (startedRef.current) return;
    startedRef.current = true;

    apiFetch("/api/demo/start", { method: "POST" })
      .then(() => createSession("voice"))
      .then(setSession)
      .catch((err: unknown) => {
        setError(
          err instanceof ApiError
            ? err.message
            : "We couldn't start a preview session. Please try again."
        );
      });
  }, []);

  function appendBoardUpdate(block: BoardContentBlock) {
    const seq = nextSeqRef.current++;
    setEvents((prev) => [
      ...prev,
      { seq, type: "board_update", role: "assistant", board: block, timestamp: new Date().toISOString() },
    ]);
  }

  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="flex max-w-sm flex-col items-center gap-3 text-center">
          <AlertCircle className="h-6 w-6 text-[var(--color-danger)]" />
          <p className="text-sm text-[var(--color-text-secondary)]">{error}</p>
          <Link
            href="/signup"
            className="rounded-[var(--radius-md)] bg-[var(--color-accent)] px-4 py-2 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90"
          >
            Sign up instead
          </Link>
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex flex-1 items-center justify-center py-16">
        <span className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
          <Loader2 className="h-4 w-4 animate-spin" />
          Starting your preview…
        </span>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
      <div className="h-3/5 w-full shrink-0 overflow-y-auto border-b border-[var(--color-border)] lg:h-full lg:w-auto lg:flex-1 lg:border-b-0">
        <Board events={events} audioSync={audioSync} />
      </div>
      <div className="flex min-h-0 flex-1 flex-col lg:w-96 lg:flex-none lg:border-l lg:border-[var(--color-border)]">
        <div className="min-h-0 flex-1 overflow-y-auto">
          <VoiceControls
            sessionId={session.id}
            onBoardUpdate={appendBoardUpdate}
            audioSync={audioSync}
            demoMode
          />
        </div>
      </div>
    </div>
  );
}
