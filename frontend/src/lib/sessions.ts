import { apiFetch } from "@/lib/api";
import { TutorSession } from "@/types/board";

export async function createSession(
  mode: "text" | "voice",
  documentIds?: string[]
): Promise<TutorSession> {
  return apiFetch<TutorSession>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({ mode, ...(documentIds ? { document_ids: documentIds } : {}) }),
  });
}
