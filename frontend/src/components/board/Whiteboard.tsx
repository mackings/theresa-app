import { Segment } from "@/lib/board/segments";
import { useTypewriter } from "@/lib/board/useTypewriter";
import { BoardCaret } from "@/components/board/BoardCaret";
import { caveat } from "@/components/board/fonts";

function SegmentView({ segment }: { segment: Segment }) {
  switch (segment.type) {
    case "math":
      return <span dangerouslySetInnerHTML={{ __html: segment.html }} />;
    case "code-inline":
      return (
        <code className="rounded bg-[var(--color-surface-hover)] px-1.5 py-0.5 font-mono text-[0.9em]">
          {segment.content}
        </code>
      );
    case "code-block":
      return (
        <pre className="my-1 overflow-x-auto rounded-lg bg-[var(--color-surface-hover)] p-3 font-mono text-sm">
          <code>{segment.content}</code>
        </pre>
      );
    case "text":
    default:
      return <span>{segment.content}</span>;
  }
}

export function Whiteboard({
  title,
  lines,
  onComplete,
}: {
  title?: string;
  lines: string[];
  onComplete: () => void;
}) {
  const { visibleLines, done } = useTypewriter(lines, onComplete);

  return (
    <div className={`${caveat.variable} px-6 py-6`}>
      {title && (
        <p className="mb-3 font-mono text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
          {title}
        </p>
      )}
      <div className="space-y-2 text-xl leading-relaxed text-[var(--color-text-primary)]" style={{ fontFamily: "var(--font-caveat), cursive" }}>
        {visibleLines.map((segments, i) => {
          const isActiveLine = i === visibleLines.length - 1;
          return (
            <p key={i}>
              {segments.map((segment, j) => (
                <SegmentView key={j} segment={segment} />
              ))}
              {isActiveLine && !done && <BoardCaret />}
            </p>
          );
        })}
      </div>
    </div>
  );
}
