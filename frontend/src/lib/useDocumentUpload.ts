import { useEffect, useRef, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
import { DocumentMeta } from "@/types/board";

const POLL_INTERVAL_MS = 2000;
// A single failed poll request (a mobile network blip, a brief backend
// hiccup) doesn't mean anything actually went wrong - the document may well
// keep processing, or already have finished, server-side regardless. Only
// give up after several in a row genuinely fail, instead of letting one
// glitch abort the whole flow and misreport a real upload as failed.
const MAX_CONSECUTIVE_POLL_FAILURES = 3;
// Slightly above the backend's own 2-minute processing budget
// (document_handlers.go's processDocument) - if it hasn't resolved by then
// something is genuinely stuck, not just slow.
const MAX_POLL_DURATION_MS = 150_000;

// Shared upload+poll logic behind both the dashboard's full dropzone panel
// and the chat composer's compact upload icon - same flow, two different
// visual presentations.
export function useDocumentUpload(onDocumentReady: (doc: DocumentMeta) => void) {
  const [doc, setDoc] = useState<DocumentMeta | null>(null);
  const [error, setError] = useState<string | null>(null);
  const cancelledRef = useRef(false);

  useEffect(() => {
    cancelledRef.current = false;
    return () => {
      cancelledRef.current = true;
    };
  }, []);

  async function pollUntilDone(id: string) {
    const deadline = Date.now() + MAX_POLL_DURATION_MS;
    let consecutiveFailures = 0;

    while (!cancelledRef.current) {
      await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
      if (cancelledRef.current) return;

      let updated: DocumentMeta;
      try {
        updated = await apiFetch<DocumentMeta>(`/api/documents/${id}`);
      } catch {
        consecutiveFailures++;
        if (consecutiveFailures < MAX_CONSECUTIVE_POLL_FAILURES) continue;
        setError(
          "Your file uploaded, but we lost connection while checking on it. " +
            "It may still finish processing - reopen this to check, or try again."
        );
        return;
      }
      consecutiveFailures = 0;
      if (cancelledRef.current) return;

      setDoc(updated);
      if (updated.status === "understood") {
        onDocumentReady(updated);
        return;
      }
      if (updated.status === "failed") {
        setError(updated.error_message ?? "We couldn't process this document.");
        return;
      }
      if (Date.now() > deadline) {
        setError(
          "This is taking longer than expected. It may still finish in the " +
            "background - reopen this to check, or try a smaller file."
        );
        return;
      }
    }
  }

  async function onFileSelected(file: File) {
    setError(null);
    setDoc(null);

    const formData = new FormData();
    formData.append("file", file);

    let uploaded: DocumentMeta;
    try {
      uploaded = await apiFetch<DocumentMeta>("/api/documents", {
        method: "POST",
        body: formData,
      });
    } catch (err) {
      // Only a failure of the initial upload request itself lands here -
      // pollUntilDone handles its own errors and never throws, so this
      // can no longer be misattributed to an unrelated later polling hiccup.
      setError(
        err instanceof ApiError ? err.message : "Upload failed - check your connection and try again."
      );
      return;
    }

    setDoc(uploaded);
    await pollUntilDone(uploaded.id);
  }

  // Reuses a document that's already been uploaded and understood, skipping
  // the upload+processing round trip entirely - same terminal state
  // (`doc` set, `onDocumentReady` fired) a fresh upload reaches, so every
  // caller's existing "processing"/"understood" rendering just works
  // unchanged regardless of which path got there.
  function selectExisting(existing: DocumentMeta) {
    setError(null);
    setDoc(existing);
    onDocumentReady(existing);
  }

  function reset() {
    setDoc(null);
    setError(null);
  }

  return { doc, error, onFileSelected, selectExisting, reset };
}
