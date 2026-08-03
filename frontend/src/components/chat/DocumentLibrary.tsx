"use client";

import { useEffect, useState } from "react";
import { FileText, Loader2 } from "lucide-react";
import { listDocuments } from "@/lib/documents";
import { DocumentMeta } from "@/types/board";

function relativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diffMs / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

// A student's course material rarely changes turn to turn - re-uploading
// the exact same PDF for every new session (observed happening repeatedly
// in production) wastes both the upload step and a real Gemini processing
// call for a file we already understood. This lists what's already been
// processed so a session/message can just reference it again instead.
export function DocumentLibrary({ onSelect }: { onSelect: (doc: DocumentMeta) => void }) {
  const [docs, setDocs] = useState<DocumentMeta[] | null>(null);

  useEffect(() => {
    listDocuments()
      .then((all) => setDocs(all.filter((d) => d.status === "understood")))
      .catch(() => setDocs([]));
  }, []);

  if (docs === null) {
    return (
      <p className="flex items-center gap-1.5 px-1 py-2 text-xs text-[var(--color-text-secondary)]">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Loading your materials...
      </p>
    );
  }

  if (docs.length === 0) {
    return (
      <p className="px-1 py-2 text-xs text-[var(--color-text-secondary)]">
        You haven&apos;t uploaded any materials yet.
      </p>
    );
  }

  return (
    <div className="max-h-48 space-y-0.5 overflow-y-auto">
      {docs.map((doc) => (
        <button
          key={doc.id}
          type="button"
          onClick={() => onSelect(doc)}
          className="flex w-full items-center gap-2 rounded-[var(--radius-md)] px-2 py-2 text-left transition-colors hover:bg-[var(--color-surface-hover)]"
        >
          <FileText className="h-3.5 w-3.5 shrink-0 text-[var(--color-text-secondary)]" />
          <span className="min-w-0 flex-1 truncate text-sm text-[var(--color-text-primary)]">
            {doc.filename}
          </span>
          <span className="shrink-0 text-xs text-[var(--color-text-secondary)]">
            {relativeTime(doc.created_at)}
          </span>
        </button>
      ))}
    </div>
  );
}
