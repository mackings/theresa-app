"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch, streamSessionMessage } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Board } from "@/components/board/Board";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { VoiceControls } from "@/components/voice/VoiceControls";
import { ModeSwitch } from "@/components/session/ModeSwitch";
import { BoardAudioSync } from "@/lib/board/audioSync";
import { TutorSession, SessionEvent, DocumentMeta, BoardContentBlock } from "@/types/board";

export default function SessionPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [session, setSession] = useState<TutorSession | null>(null);
  const [mode, setMode] = useState<"text" | "voice">("text");
  const [switchingMode, setSwitchingMode] = useState(false);
  const [events, setEvents] = useState<SessionEvent[]>([]);
  const [initialEventCount, setInitialEventCount] = useState(0);
  const [documentId, setDocumentId] = useState<string | undefined>(undefined);
  const [solving, setSolving] = useState(false);
  const nextSeqRef = useRef(0);
  const [audioSync] = useState(() => new BoardAudioSync());

  useEffect(() => {
    apiFetch<TutorSession>(`/api/sessions/${params.id}`)
      .then((s) => {
        setSession(s);
        setMode(s.mode);
        const initialEvents = s.events ?? [];
        setEvents(initialEvents);
        setInitialEventCount(initialEvents.length);
        nextSeqRef.current = initialEvents.length;

        // The dashboard's composer/upload flow hands off a message here
        // instead of sending it itself and waiting - that would leave the
        // user staring at a disabled button for the whole Gemini call, then
        // drop them onto an already-fully-solved board with no chance to see
        // it "start solving" at all. Sending it here instead means the
        // board's pending/thinking state (below) is what the user actually
        // watches. Removed from storage as the very first thing so a
        // duplicate effect run (React Strict Mode's dev-only
        // mount->cleanup->remount double-invoke) can't fire it twice.
        const key = `theresa:pending-message:${s.id}`;
        const raw = sessionStorage.getItem(key);
        if (!raw) return;
        sessionStorage.removeItem(key);

        let payload: { text: string; documentId?: string };
        try {
          payload = JSON.parse(raw);
        } catch {
          return;
        }

        setSolving(true);
        return streamSessionMessage(
          s.id,
          { text: payload.text, ...(payload.documentId ? { document_id: payload.documentId } : {}) },
          (event) => setEvents((prev) => [...prev, event])
        ).finally(() => setSolving(false));
      })
      .catch(() => router.replace("/dashboard"));
  }, [params.id, router]);

  if (!session) {
    return null;
  }

  function appendBoardUpdate(block: BoardContentBlock) {
    const seq = nextSeqRef.current++;
    setEvents((prev) => [
      ...prev,
      { seq, type: "board_update", role: "assistant", board: block, timestamp: new Date().toISOString() },
    ]);
  }

  async function handleModeChange(next: "text" | "voice") {
    if (next === mode || switchingMode) return;
    setSwitchingMode(true);
    try {
      await apiFetch(`/api/sessions/${session!.id}`, {
        method: "PATCH",
        body: JSON.stringify({ mode: next }),
      });
      setMode(next);
    } finally {
      setSwitchingMode(false);
    }
  }

  return (
    <AppShell>
      <div className="flex h-full flex-col lg:flex-row">
        <div className="h-3/5 w-full shrink-0 overflow-y-auto border-b border-[var(--color-border)] lg:h-full lg:w-auto lg:flex-1 lg:border-b-0">
          <Board
            events={events}
            audioSync={mode === "voice" ? audioSync : undefined}
            pending={mode === "text" && solving}
            instantUpToSeq={initialEventCount}
          />
        </div>
        <div className="flex min-h-0 flex-1 flex-col lg:w-96 lg:flex-none lg:border-l lg:border-[var(--color-border)]">
          <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2.5">
            <p className="truncate text-sm font-medium text-[var(--color-text-primary)]">
              {session.title}
            </p>
            <ModeSwitch mode={mode} onChange={handleModeChange} disabled={switchingMode} />
          </div>
          {mode === "voice" ? (
            <div className="min-h-0 flex-1 overflow-y-auto">
              <VoiceControls
                sessionId={session.id}
                onBoardUpdate={appendBoardUpdate}
                audioSync={audioSync}
              />
            </div>
          ) : (
            <div className="flex-1 overflow-hidden">
              <ChatPanel
                sessionId={session.id}
                events={events}
                documentId={documentId}
                onDocumentReady={(doc: DocumentMeta) => setDocumentId(doc.id)}
                onNewEvents={(newEvents) => setEvents((prev) => [...prev, ...newEvents])}
                solving={solving}
                onSolvingChange={setSolving}
              />
            </div>
          )}
        </div>
      </div>
    </AppShell>
  );
}
