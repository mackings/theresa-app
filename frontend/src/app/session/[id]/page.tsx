"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Board } from "@/components/board/Board";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { UploadDropzone } from "@/components/chat/UploadDropzone";
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
  const [documentId, setDocumentId] = useState<string | undefined>(undefined);
  const nextSeqRef = useRef(0);
  const [audioSync] = useState(() => new BoardAudioSync());

  useEffect(() => {
    apiFetch<TutorSession>(`/api/sessions/${params.id}`)
      .then((s) => {
        setSession(s);
        setMode(s.mode);
        const initialEvents = s.events ?? [];
        setEvents(initialEvents);
        nextSeqRef.current = initialEvents.length;
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
      <div className="flex h-full">
        <div className="flex-1 overflow-y-auto">
          <Board events={events} audioSync={mode === "voice" ? audioSync : undefined} />
        </div>
        <div className="flex w-96 shrink-0 flex-col border-l border-[var(--color-border)]">
          <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2.5">
            <p className="truncate text-sm font-medium text-[var(--color-text-primary)]">
              {session.title}
            </p>
            <ModeSwitch mode={mode} onChange={handleModeChange} disabled={switchingMode} />
          </div>
          {mode === "voice" ? (
            <VoiceControls
              sessionId={session.id}
              onBoardUpdate={appendBoardUpdate}
              audioSync={audioSync}
            />
          ) : (
            <>
              <UploadDropzone onDocumentReady={(doc: DocumentMeta) => setDocumentId(doc.id)} />
              <div className="flex-1 overflow-hidden">
                <ChatPanel
                  sessionId={session.id}
                  events={events}
                  documentId={documentId}
                  onNewEvents={(newEvents) => setEvents((prev) => [...prev, ...newEvents])}
                />
              </div>
            </>
          )}
        </div>
      </div>
    </AppShell>
  );
}
