import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, PenLine } from "lucide-react";
import { SessionEvent } from "@/types/board";
import { Whiteboard } from "@/components/board/Whiteboard";
import { DiagramBoard } from "@/components/board/DiagramBoard";
import type { BoardAudioSync } from "@/lib/board/audioSync";

function isRenderableBoardEvent(e: SessionEvent): boolean {
  return (
    e.type === "board_update" &&
    !!e.board &&
    (e.board.kind === "lines" || e.board.kind === "diagram" || e.board.kind === "clear")
  );
}

function noop() {}

// A "clear" event (Theresa erasing the board on request) has no content of
// its own to reveal - it's purely a reset signal. Only what comes after the
// LAST clear in a batch is ever worth showing (anything before it is
// guaranteed superseded), so this strips a clear and everything ahead of it
// down to nothing, reporting that clear's seq back so the caller can wipe
// whatever was already accumulated and fast-forward the cursor past it.
function dropUpToLastClear(events: SessionEvent[]): { events: SessionEvent[]; lastClearSeq: number | null } {
  let lastClearIdx = -1;
  for (let i = 0; i < events.length; i++) {
    if (events[i].board?.kind === "clear") lastClearIdx = i;
  }
  if (lastClearIdx === -1) return { events, lastClearSeq: null };
  return { events: events.slice(lastClearIdx + 1), lastClearSeq: events[lastClearIdx].seq };
}

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
  instantUpToSeq = 0,
}: {
  events: SessionEvent[];
  audioSync?: BoardAudioSync;
  pending?: boolean;
  // Events with seq below this were already taught before this visit -
  // reopening a session should show that history instantly, not replay the
  // whole typewriter sequence again. Only genuinely new events (seq at or
  // above this) get the live reveal animation.
  instantUpToSeq?: number;
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

    const pendingRaw = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    if (pendingRaw.length === 0) return;

    const { events: pendingEvents, lastClearSeq } = dropUpToLastClear(pendingRaw);
    if (lastClearSeq !== null) {
      setVisibleEvents([]);
      cursorSeqRef.current = lastClearSeq;
    }
    if (pendingEvents.length === 0) return;

    // Everything already taught before this visit lands in one batch -
    // reveal it all instantly instead of one at a time. Anything genuinely
    // new (this visit) still goes through the normal reveal-then-advance
    // cycle below.
    const historical = pendingEvents.filter((e) => e.seq < instantUpToSeq);
    const live = pendingEvents.filter((e) => e.seq >= instantUpToSeq);

    if (historical.length > 0) {
      cursorSeqRef.current = historical[historical.length - 1].seq;
      setVisibleEvents((prev) => [...prev, ...historical]);
    }

    if (live.length > 0) {
      const next = live[0];
      activeSeqRef.current = next.seq;
      setActiveSeq(next.seq);
      setVisibleEvents((prev) => [...prev, next]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boardEventsKey]);

  function handleActiveComplete() {
    if (activeSeqRef.current !== null) {
      cursorSeqRef.current = activeSeqRef.current;
    }
    audioSync?.markDone();
    activeSeqRef.current = null;
    setActiveSeq(null);

    const pendingRaw = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    if (pendingRaw.length === 0) return;

    const { events: pending, lastClearSeq } = dropUpToLastClear(pendingRaw);
    if (lastClearSeq !== null) {
      setVisibleEvents([]);
      cursorSeqRef.current = lastClearSeq;
    }
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
                  instant={event.seq < instantUpToSeq}
                />
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
