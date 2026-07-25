import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, PenLine } from "lucide-react";
import { SessionEvent } from "@/types/board";
import { Whiteboard } from "@/components/board/Whiteboard";
import { DiagramBoard } from "@/components/board/DiagramBoard";
import type { BoardAudioSync } from "@/lib/board/audioSync";

function isRenderableBoardEvent(e: SessionEvent): boolean {
  return e.type === "board_update" && !!e.board && (e.board.kind === "lines" || e.board.kind === "diagram");
}

function noop() {}

// Board accumulates every unit as it arrives instead of wiping between them:
// each new board is typed out and stays on screen, growing downward (with
// Whiteboard/DiagramBoard's own auto-scroll keeping the newest content in
// view), like a real notebook page filling up rather than a slideshow.
// Boards still reveal one at a time, in seq order - a fast text-mode stream
// can easily produce a new step before the previous one has finished
// typewriting, so a newly-arrived unit just waits its turn instead of
// interrupting whatever's currently revealing.
export function Board({
  events,
  audioSync,
  pending,
}: {
  events: SessionEvent[];
  audioSync?: BoardAudioSync;
  pending?: boolean;
}) {
  const boardEvents = useMemo(
    () => events.filter(isRenderableBoardEvent).sort((a, b) => a.seq - b.seq),
    [events]
  );
  const boardEventsKey = boardEvents.map((e) => e.seq).join(",");

  const [visibleEvents, setVisibleEvents] = useState<SessionEvent[]>([]);
  const [activeSeq, setActiveSeq] = useState<number | null>(null);

  const cursorSeqRef = useRef(-1);
  const activeSeqRef = useRef<number | null>(null);
  const processedKeyRef = useRef<string | null>(null);

  useEffect(() => {
    // React Strict Mode double-invokes effects in dev - since the refs this
    // effect mutates survive that synthetic replay, a second run with the
    // exact same boardEventsKey would otherwise misread its own first run's
    // ref update as "new content arrived." Guard against reprocessing a key
    // that hasn't actually changed.
    if (processedKeyRef.current === boardEventsKey) return;
    processedKeyRef.current = boardEventsKey;

    if (activeSeqRef.current !== null) {
      // Something's already revealing - whatever's newly pending gets picked
      // up by handleActiveComplete once it finishes. Boards accumulate, they
      // never interrupt one that's mid-reveal.
      return;
    }

    const pending = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    if (pending.length === 0) return;

    const next = pending[0];
    activeSeqRef.current = next.seq;
    setActiveSeq(next.seq);
    setVisibleEvents((prev) => [...prev, next]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boardEventsKey]);

  function handleActiveComplete() {
    if (activeSeqRef.current !== null) {
      cursorSeqRef.current = activeSeqRef.current;
    }
    audioSync?.markDone();
    activeSeqRef.current = null;
    setActiveSeq(null);

    const pending = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    if (pending.length === 0) return;

    const next = pending[0];
    activeSeqRef.current = next.seq;
    setActiveSeq(next.seq);
    setVisibleEvents((prev) => [...prev, next]);
  }

  useEffect(() => {
    // A fresh board started revealing - this is when typing actually starts,
    // so the audio-pacing anchor resets here.
    if (activeSeq !== null) {
      audioSync?.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSeq]);

  if (visibleEvents.length === 0) {
    return (
      <div className="flex h-full items-center justify-center bg-[var(--color-bg)] px-6 text-sm text-[var(--color-text-secondary)]">
        <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border)] bg-[var(--color-surface)] px-8 py-10 text-center">
          {pending ? (
            <span className="inline-flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Theresa is working on it…
            </span>
          ) : (
            "Paste a problem or upload a document to get started."
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto bg-[var(--color-bg)] px-6 py-10">
      <div className="mx-auto w-full max-w-2xl">
        <div className="min-h-[420px] rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-lg)]">
          <div className="flex items-center justify-between border-b border-[var(--color-border-subtle)] px-4 py-2.5 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
            <span className="flex items-center gap-1.5">
              <PenLine className="h-3.5 w-3.5" />
              Board
            </span>
            {pending && (
              <span className="flex items-center gap-1.5 normal-case tracking-normal text-[var(--color-text-secondary)]">
                <Loader2 className="h-3 w-3 animate-spin" />
                Thinking…
              </span>
            )}
          </div>
          <div className="divide-y divide-[var(--color-border-subtle)]">
            {visibleEvents.map((event) => {
              const board = event.board!;
              const isActive = event.seq === activeSeq;
              return board.kind === "diagram" ? (
                <DiagramBoard
                  key={event.seq}
                  title={board.title}
                  mermaid={board.mermaid ?? ""}
                  onComplete={isActive ? handleActiveComplete : noop}
                />
              ) : (
                <Whiteboard
                  key={event.seq}
                  title={board.title}
                  lines={board.lines ?? []}
                  onComplete={isActive ? handleActiveComplete : noop}
                />
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
