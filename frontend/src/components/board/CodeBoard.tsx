"use client";

import { useEffect, useRef, useState } from "react";
import CodeMirror, { Extension } from "@uiw/react-codemirror";
import { oneDark } from "@codemirror/theme-one-dark";
import { CODE_DWELL_MS } from "@/lib/board/timing";

// Same small alias set as the backend's normalizeCodeLanguage (tools.go) -
// kept independent rather than shared, since the backend's job is picking a
// canonical key and this one's job is picking a loadable CodeMirror
// extension; an unrecognized language here just falls back to no extension
// (plain highlighted text), never throws.
async function loadLanguageExtension(language: string): Promise<Extension | null> {
  switch (language) {
    case "javascript":
    case "jsx":
      return (await import("@codemirror/lang-javascript")).javascript({ jsx: true });
    case "typescript":
    case "tsx":
      return (await import("@codemirror/lang-javascript")).javascript({ jsx: true, typescript: true });
    case "python":
      return (await import("@codemirror/lang-python")).python();
    default:
      return null;
  }
}

// CodeBoard renders Theresa's own code demonstrations - read-only, real
// syntax highlighting via CodeMirror. Same onComplete contract as
// DiagramBoard: a ref-stabilized onCompleteRef (so a parent re-render can
// never reset an in-progress reveal), a fixed dwell after rendering before
// calling onComplete. Unlike Whiteboard's per-character typewriter,
// CodeMirror's own editor state doesn't compose with revealing content
// letter-by-letter, so this renders its content whole immediately and only
// paces the dwell before the board advances - the same shape DiagramBoard
// already uses for its own not-really-typewriter-able content (an SVG).
export function CodeBoard({
  title,
  code,
  language,
  onComplete,
}: {
  title?: string;
  code: string;
  language?: string;
  onComplete: () => void;
}) {
  const [extension, setExtension] = useState<Extension | null>(null);

  const onCompleteRef = useRef(onComplete);
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    let cancelled = false;
    loadLanguageExtension(language ?? "").then((ext) => {
      if (!cancelled) setExtension(ext);
    });

    const timeoutId = setTimeout(() => {
      if (!cancelled) onCompleteRef.current();
    }, CODE_DWELL_MS);

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
    };
  }, [code, language]);

  return (
    <div className="px-6 py-6">
      {title && (
        <p className="mb-3 font-mono text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
          {title}
        </p>
      )}
      <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-subtle)]">
        <CodeMirror
          value={code}
          editable={false}
          theme={oneDark}
          extensions={extension ? [extension] : []}
          basicSetup={{ lineNumbers: true, foldGutter: false, highlightActiveLine: false }}
        />
      </div>
    </div>
  );
}
