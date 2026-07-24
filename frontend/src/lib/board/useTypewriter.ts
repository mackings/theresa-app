import { useEffect, useRef, useState } from "react";
import { parseLine, Segment } from "./segments";
import { CHAR_REVEAL_MS, MATH_DWELL_MS, LINE_SETTLE_MS } from "./timing";

export interface TypewriterState {
  visibleLines: Segment[][];
  done: boolean;
}

// useTypewriter reveals `lines` at a fixed pace: text/code characters one at
// a time (CHAR_REVEAL_MS apart), math segments as one whole pre-rendered
// unit followed by a dwell (MATH_DWELL_MS), and a settle pause
// (LINE_SETTLE_MS) between finished lines. No pause after the last line -
// onComplete fires the instant its last unit is shown.
//
// Content is parsed once per mount (lines are assumed fixed for this
// instance's lifetime - Board.tsx remounts via `key` whenever the actual
// content changes, rather than this hook trying to detect content changes
// itself). onComplete is stored in a ref and never appears in the reveal
// effect's dependency array, so an unrelated parent re-render can't restart
// or interrupt an in-progress reveal.
export function useTypewriter(lines: string[], onComplete: () => void): TypewriterState {
  const parsedLinesRef = useRef<Segment[][]>(undefined);
  if (parsedLinesRef.current === undefined) {
    parsedLinesRef.current = lines.map(parseLine);
  }

  const onCompleteRef = useRef(onComplete);
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  const [state, setState] = useState<TypewriterState>({
    visibleLines: [],
    done: lines.length === 0,
  });

  useEffect(() => {
    const parsedLines = parsedLinesRef.current!;

    if (parsedLines.length === 0) {
      onCompleteRef.current();
      return;
    }

    let cancelled = false;
    let timeoutId: ReturnType<typeof setTimeout>;
    let lineIdx = 0;
    let segIdx = 0;
    let charIdx = 0;
    let mathRevealed = false;

    function render() {
      if (cancelled) return;

      const visibleLines: Segment[][] = parsedLines.slice(0, lineIdx);
      const segs = parsedLines[lineIdx] ?? [];
      const activeLineSegs: Segment[] = segs.slice(0, segIdx);
      const cur = segs[segIdx];

      if (cur) {
        if (cur.type === "math") {
          if (mathRevealed) activeLineSegs.push(cur);
        } else if (charIdx > 0) {
          activeLineSegs.push({ ...cur, content: cur.content.slice(0, charIdx) } as Segment);
        }
      }

      if (activeLineSegs.length > 0 || lineIdx < parsedLines.length) {
        visibleLines.push(activeLineSegs);
      }

      setState({ visibleLines, done: false });
    }

    function scheduleNext(delay: number) {
      timeoutId = setTimeout(tick, delay);
    }

    function tick() {
      if (cancelled) return;
      const segs = parsedLines[lineIdx] ?? [];

      if (segIdx >= segs.length) {
        lineIdx += 1;
        segIdx = 0;
        charIdx = 0;
        mathRevealed = false;

        if (lineIdx >= parsedLines.length) {
          render();
          setState((s) => ({ ...s, done: true }));
          onCompleteRef.current();
          return;
        }

        render();
        scheduleNext(LINE_SETTLE_MS);
        return;
      }

      const cur = segs[segIdx];

      if (cur.type === "math") {
        mathRevealed = true;
        render();
        segIdx += 1;
        mathRevealed = false;
        scheduleNext(MATH_DWELL_MS);
        return;
      }

      charIdx += 1;
      render();
      if (charIdx >= cur.content.length) {
        segIdx += 1;
        charIdx = 0;
      }
      scheduleNext(CHAR_REVEAL_MS);
    }

    render();
    scheduleNext(0);

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
    };
  }, []);

  return state;
}
