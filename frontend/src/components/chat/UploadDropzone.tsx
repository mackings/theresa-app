"use client";

import { useRef, useState } from "react";
import { AlertCircle, FileCheck2, FolderOpen, Loader2, RotateCcw, Upload } from "lucide-react";
import { IconButton } from "@/components/ui/Button";
import { useDocumentUpload } from "@/lib/useDocumentUpload";
import { DocumentLibrary } from "@/components/chat/DocumentLibrary";
import { DocumentMeta } from "@/types/board";

export function UploadDropzone({
  onDocumentReady,
}: {
  onDocumentReady: (doc: DocumentMeta) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);
  const [showLibrary, setShowLibrary] = useState(false);
  const { doc, error, uploadingFilename, onFileSelected, selectExisting, reset } =
    useDocumentUpload(onDocumentReady);

  return (
    <div className="border-t border-[var(--color-border)] p-3">
      <input
        ref={inputRef}
        type="file"
        accept=".pdf,.jpg,.jpeg,.png"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) onFileSelected(file);
        }}
      />

      {!doc && !error && !uploadingFilename && (
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragOver(false);
            const file = e.dataTransfer.files?.[0];
            if (file) onFileSelected(file);
          }}
          className={`flex w-full flex-col items-center justify-center gap-1.5 rounded-[var(--radius-md)] border border-dashed px-3 py-4 text-sm transition-colors ${
            dragOver
              ? "border-[var(--color-accent)] bg-[var(--color-surface-hover)] text-[var(--color-text-primary)]"
              : "border-[var(--color-border)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)]"
          }`}
        >
          <Upload className="h-4 w-4" />
          Upload a PDF or photo of a page
          <span className="text-xs text-[var(--color-text-secondary)]">
            PDF, JPG, or PNG - up to 20MB
          </span>
        </button>
      )}

      {!doc && !error && !uploadingFilename && (
        <button
          type="button"
          onClick={() => setShowLibrary((v) => !v)}
          className="mt-2 flex w-full items-center justify-center gap-1.5 rounded-[var(--radius-md)] px-2 py-1.5 text-xs text-[var(--color-text-secondary)] transition-colors hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)]"
        >
          <FolderOpen className="h-3.5 w-3.5" />
          {showLibrary ? "Hide" : "Or reuse something you've already uploaded"}
        </button>
      )}
      {!doc && !error && !uploadingFilename && showLibrary && (
        <div className="mt-1">
          <DocumentLibrary onSelect={selectExisting} />
        </div>
      )}

      {uploadingFilename && (
        <p className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Uploading {uploadingFilename}...
        </p>
      )}

      {doc && doc.status === "processing" && (
        <p className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Processing {doc.filename}...
        </p>
      )}

      {doc && doc.status === "understood" && (
        <div className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
          <FileCheck2 className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--color-accent)]" />
          <div>
            <p className="font-medium text-[var(--color-text-primary)]">{doc.filename}</p>
            <p className="mt-1">{doc.extracted_summary}</p>
          </div>
        </div>
      )}

      {error && (
        <div className="flex items-center justify-between gap-2 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-2 text-xs text-[var(--color-danger)]">
          <span className="flex items-center gap-1.5">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
            {error}
          </span>
          <IconButton
            variant="ghost"
            aria-label="Try again"
            onClick={reset}
            className="h-6 w-6"
          >
            <RotateCcw className="h-3.5 w-3.5" />
          </IconButton>
        </div>
      )}
    </div>
  );
}
