import katex from "katex";

export type Segment =
  | { type: "text"; content: string }
  | { type: "code-inline"; content: string }
  | { type: "code-block"; content: string; language?: string }
  | { type: "math"; source: string; html: string };

const FENCE = "```";

function renderMath(tex: string): string {
  try {
    return katex.renderToString(tex, { throwOnError: false, errorColor: "#cc0000" });
  } catch {
    return tex;
  }
}

// parseLine turns one board "line" into an ordered list of segments. A line
// that is itself a whole fenced code block (```...```) is treated as one
// atomic code-block segment - Gemini emits the full fence, including
// embedded newlines, as a single array entry. Otherwise, inline math
// ($...$) and inline code (`...`) are pulled out in a single linear pass
// so one delimiter type can't accidentally re-match inside the other's
// already-matched span.
export function parseLine(line: string): Segment[] {
  const trimmed = line.trim();
  if (trimmed.startsWith(FENCE) && trimmed.endsWith(FENCE) && trimmed.length > FENCE.length * 2) {
    const inner = trimmed.slice(FENCE.length, -FENCE.length);
    const firstNewline = inner.indexOf("\n");
    const language = firstNewline === -1 ? undefined : inner.slice(0, firstNewline).trim() || undefined;
    const content = firstNewline === -1 ? inner : inner.slice(firstNewline + 1);
    return [{ type: "code-block", content, language }];
  }

  const segments: Segment[] = [];
  const pattern = /\$([^$]+)\$|`([^`]+)`/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(line)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ type: "text", content: line.slice(lastIndex, match.index) });
    }
    if (match[1] !== undefined) {
      segments.push({ type: "math", source: match[1], html: renderMath(match[1]) });
    } else if (match[2] !== undefined) {
      segments.push({ type: "code-inline", content: match[2] });
    }
    lastIndex = pattern.lastIndex;
  }

  if (lastIndex < line.length) {
    segments.push({ type: "text", content: line.slice(lastIndex) });
  }

  return segments.length > 0 ? segments : [{ type: "text", content: line }];
}
