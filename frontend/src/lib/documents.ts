import { apiFetch } from "@/lib/api";
import { DocumentMeta } from "@/types/board";

export function listDocuments(): Promise<DocumentMeta[]> {
  return apiFetch<DocumentMeta[]>("/api/documents");
}
