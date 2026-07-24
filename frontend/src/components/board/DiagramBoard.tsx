"use client";

import { useEffect, useId, useRef, useState } from "react";
import { DIAGRAM_DWELL_MS } from "@/lib/board/timing";

export function DiagramBoard({
  title,
  mermaid: spec,
  onComplete,
}: {
  title?: string;
  mermaid: string;
  onComplete: () => void;
}) {
  const id = useId().replace(/:/g, "-");
  const [svg, setSvg] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  const onCompleteRef = useRef(onComplete);
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    let cancelled = false;
    let timeoutId: ReturnType<typeof setTimeout>;

    async function render() {
      try {
        const mermaid = (await import("mermaid")).default;
        mermaid.initialize({ startOnLoad: false, theme: "neutral" });
        const { svg } = await mermaid.render(`mermaid-${id}`, spec);
        if (cancelled) return;
        setSvg(svg);
        // Diagrams appear fully-formed (no per-character reveal is possible
        // for an SVG) - hold for a fixed dwell so narration gets a beat to
        // let the student actually look at it before the board advances.
        timeoutId = setTimeout(() => {
          if (!cancelled) onCompleteRef.current();
        }, DIAGRAM_DWELL_MS);
      } catch {
        if (cancelled) return;
        setFailed(true);
        // No content worth holding on a failure - advance immediately
        // rather than burning the dwell on an error message.
        onCompleteRef.current();
      }
    }

    render();
    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
    };
  }, [spec, id]);

  return (
    <div className="px-6 py-6">
      {title && (
        <p className="mb-3 font-mono text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
          {title}
        </p>
      )}

      {failed && (
        <p className="text-sm text-[var(--color-text-secondary)]">
          Couldn&apos;t draw this diagram.
        </p>
      )}

      {!failed && !svg && (
        <div className="h-24 animate-pulse rounded-lg bg-[var(--color-surface-hover)]" />
      )}

      {!failed && svg && (
        <div className="overflow-x-auto" dangerouslySetInnerHTML={{ __html: svg }} />
      )}
    </div>
  );
}
