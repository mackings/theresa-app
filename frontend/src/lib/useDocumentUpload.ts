import { useEffect, useRef, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
import { DocumentMeta } from "@/types/board";

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
    while (!cancelledRef.current) {
      await new Promise((r) => setTimeout(r, 2000));
      if (cancelledRef.current) return;

      const updated = await apiFetch<DocumentMeta>(`/api/documents/${id}`);
      if (cancelledRef.current) return;

      setDoc(updated);
      if (updated.status === "understood") {
        onDocumentReady(updated);
        return;
      }
      if (updated.status === "failed") {
        setError(updated.error_message ?? "failed to process document");
        return;
      }
    }
  }

  async function onFileSelected(file: File) {
    setError(null);
    setDoc(null);

    const formData = new FormData();
    formData.append("file", file);

    try {
      const uploaded = await apiFetch<DocumentMeta>("/api/documents", {
        method: "POST",
        body: formData,
      });
      setDoc(uploaded);
      await pollUntilDone(uploaded.id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "upload failed");
    }
  }

  function reset() {
    setDoc(null);
    setError(null);
  }

  return { doc, error, onFileSelected, reset };
}
