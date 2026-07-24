import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, PenLine } from "lucide-react";
import { SessionEvent } from "@/types/board";
import { Whiteboard } from "@/components/board/Whiteboard";
import { DiagramBoard } from "@/components/board/DiagramBoard";
import { WIPE_DURATION_MS } from "@/lib/board/timing";
import type { BoardAudioSync } from "@/lib/board/audioSync";

function isRenderableBoardEvent(e: SessionEvent): boolean {
  return e.type === "board_update" && !!e.board && (e.board.kind === "lines" || e.board.kind === "diagram");
}

// Board is a single-slot sequencer, not a growing list: it shows one board
// at a time, walking forward through boardEvents in seq order, with a brief
// wipe transition between one board and the next. Every board is shown -
// even ones that arrive while an earlier one is still mid-reveal (a fast
// streaming text answer routinely produces a new step before the previous
// one has finished typewriting/dwelling) - by always advancing to the very
// next pending unit rather than skipping ahead to whatever's newest.
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

  const [activeEvent, setActiveEvent] = useState<SessionEvent | null>(null);
  const [activeKey, setActiveKey] = useState(0);
  const [wiping, setWiping] = useState(false);

  const cursorSeqRef = useRef(-1);
  const activeSeqRef = useRef<number | null>(null);
  const wipingRef = useRef(false);
  const wipeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const processedKeyRef = useRef<string | null>(null);

  useEffect(() => {
    // React Strict Mode double-invokes effects in dev - since the refs this
    // effect mutates survive that synthetic replay, a second run with the
    // exact same boardEventsKey would otherwise misread its own first run's
    // ref update as "new content arrived while something's already active"
    // and immediately skip-ahead-wipe to the newest unit. Guard against
    // reprocessing a key that hasn't actually changed.
    if (processedKeyRef.current === boardEventsKey) return;
    processedKeyRef.current = boardEventsKey;

    function startWipeTo(target: SessionEvent) {
      wipingRef.current = true;
      setWiping(true);
      if (wipeTimeoutRef.current) clearTimeout(wipeTimeoutRef.current);
      wipeTimeoutRef.current = setTimeout(() => {
        wipingRef.current = false;
        activeSeqRef.current = target.seq;
        setActiveEvent(target);
        setActiveKey((k) => k + 1);
        setWiping(false);
      }, WIPE_DURATION_MS);
    }

    const pending = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    if (pending.length === 0) return;

    if (activeSeqRef.current === null) {
      // Nothing shown yet - start from the oldest pending unit and walk
      // forward one at a time (both a fresh session's first board and a
      // reloaded session's full batched history land here identically; a
      // batch is a sequence to play through, not a jump to its last frame).
      const first = pending[0];
      activeSeqRef.current = first.seq;
      setActiveEvent(first);
      setActiveKey((k) => k + 1);
      return;
    }

    if (wipingRef.current) {
      // A wipe toward an earlier-decided target is already in flight - let
      // it finish rather than retargeting mid-transition. Whatever arrived
      // after that target gets picked up on the next pass, once this wipe
      // completes and cursorSeq/activeSeq update.
      return;
    }

    if (pending[0].seq > activeSeqRef.current) {
      startWipeTo(pending[0]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boardEventsKey]);

  useEffect(() => {
    return () => {
      if (wipeTimeoutRef.current) clearTimeout(wipeTimeoutRef.current);
    };
  }, []);

  function handleActiveComplete() {
    if (activeSeqRef.current !== null) {
      cursorSeqRef.current = activeSeqRef.current;
    }
    audioSync?.markDone();

    const pending = boardEvents.filter((e) => e.seq > cursorSeqRef.current);
    activeSeqRef.current = null;

    if (pending.length === 0) {
      return;
    }

    // Advance to the very next unit in order (not the newest pending) - a
    // normal completion walks the sequence forward one step at a time. The
    // "skip ahead to newest" behavior is reserved for the case where new
    // content arrives while a unit is still mid-reveal (handled below).
    const target = pending[0];
    wipingRef.current = true;
    setWiping(true);
    if (wipeTimeoutRef.current) clearTimeout(wipeTimeoutRef.current);
    wipeTimeoutRef.current = setTimeout(() => {
      wipingRef.current = false;
      activeSeqRef.current = target.seq;
      setActiveEvent(target);
      setActiveKey((k) => k + 1);
      setWiping(false);
    }, WIPE_DURATION_MS);
  }

  useEffect(() => {
    // A fresh board mounted (end of wipe, or the very first board) - this is
    // when typing actually starts, so the audio-pacing anchor resets here.
    if (activeEvent) {
      audioSync?.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeKey]);

  if (!activeEvent) {
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

  const board = activeEvent.board!;

  return (
    <div className="flex h-full items-center justify-center overflow-y-auto bg-[var(--color-bg)] px-6 py-10">
      <div className="mx-auto w-full max-w-2xl">
        <div
          className="min-h-[420px] rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-lg)] transition-opacity"
          style={{ opacity: wiping ? 0 : 1, transitionDuration: `${WIPE_DURATION_MS}ms` }}
        >
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
          {!wiping &&
            (board.kind === "diagram" ? (
              <DiagramBoard
                key={activeKey}
                title={board.title}
                mermaid={board.mermaid ?? ""}
                onComplete={handleActiveComplete}
              />
            ) : (
              <Whiteboard
                key={activeKey}
                title={board.title}
                lines={board.lines ?? []}
                onComplete={handleActiveComplete}
              />
            ))}
        </div>
      </div>
    </div>
  );
}
