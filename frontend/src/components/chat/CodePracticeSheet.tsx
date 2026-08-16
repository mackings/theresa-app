"use client";

import { useState } from "react";
import { X, Send } from "lucide-react";
import CodeMirror from "@uiw/react-codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { python } from "@codemirror/lang-python";
import { oneDark } from "@codemirror/theme-one-dark";
import { Button, IconButton } from "@/components/ui/Button";

const LANGUAGES = ["plaintext", "python", "javascript", "typescript"] as const;
type PracticeLanguage = (typeof LANGUAGES)[number];

function extensionFor(language: PracticeLanguage) {
  switch (language) {
    case "python":
      return [python()];
    case "javascript":
      return [javascript({ jsx: true })];
    case "typescript":
      return [javascript({ jsx: true, typescript: true })];
    default:
      return [];
  }
}

// A wide, full-screen overlay rather than the session page's cramped 384px
// right column - real code editing needs width the existing panel doesn't
// have to spare. Doesn't call the API itself: onSend hands the wrapped
// message back to ChatPanel, which reuses its own existing
// streamSessionMessage/sending/onSolvingChange/onNewEvents plumbing rather
// than this component duplicating it.
export function CodePracticeSheet({
  defaultLanguage,
  sending,
  onSend,
  onClose,
}: {
  defaultLanguage?: string;
  sending: boolean;
  onSend: (message: string) => void;
  onClose: () => void;
}) {
  const initialLanguage = LANGUAGES.includes(defaultLanguage as PracticeLanguage)
    ? (defaultLanguage as PracticeLanguage)
    : "plaintext";
  const [language, setLanguage] = useState<PracticeLanguage>(initialLanguage);
  const [code, setCode] = useState("");
  const [note, setNote] = useState("");

  function handleSend() {
    if (!code.trim() || sending) return;
    const fence = language === "plaintext" ? "" : language;
    const message = `${note.trim() ? note.trim() + "\n\n" : ""}\`\`\`${fence}\n${code}\n\`\`\``;
    onSend(message);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        className="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-lg)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-[var(--color-border-subtle)] px-4 py-3">
          <p className="text-sm font-semibold text-[var(--color-text-primary)]">
            Practice - write code, get real feedback
          </p>
          <IconButton variant="ghost" aria-label="Close practice sheet" onClick={onClose}>
            <X className="h-4 w-4" />
          </IconButton>
        </div>

        <div className="flex items-center gap-2 border-b border-[var(--color-border-subtle)] px-4 py-2">
          <label className="text-xs text-[var(--color-text-secondary)]">Language</label>
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value as PracticeLanguage)}
            className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-xs text-[var(--color-text-primary)] outline-none"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang} value={lang}>
                {lang}
              </option>
            ))}
          </select>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <CodeMirror
            value={code}
            onChange={setCode}
            editable={!sending}
            theme={oneDark}
            height="320px"
            extensions={extensionFor(language)}
            basicSetup={{ lineNumbers: true }}
          />
        </div>

        <div className="border-t border-[var(--color-border-subtle)] p-3">
          <input
            type="text"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Optional note - what would you like feedback on?"
            className="w-full rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none"
          />
          <div className="mt-2 flex justify-end">
            <Button
              variant="primary"
              icon={<Send className="h-3.5 w-3.5" />}
              disabled={!code.trim() || sending}
              onClick={handleSend}
            >
              {sending ? "Sending…" : "Send for feedback"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
