export type BoardKind = "lines" | "diagram";

export interface BoardContentBlock {
  kind: BoardKind;
  title?: string;
  lines?: string[];
  mermaid?: string;
}

export interface SessionEvent {
  seq: number;
  type: "user_text" | "board_update" | "chat_message" | "transcript";
  role?: "user" | "assistant";
  text?: string;
  board?: BoardContentBlock;
  timestamp: string;
}

export interface TutorSession {
  id: string;
  title: string;
  document_ids?: string[];
  mode: "text" | "voice";
  status: "active" | "ended";
  created_at: string;
  updated_at: string;
  events?: SessionEvent[];
}

export interface DocumentMeta {
  id: string;
  filename: string;
  mime_type: string;
  size_bytes: number;
  status: "uploaded" | "processing" | "understood" | "failed";
  extracted_summary?: string;
  created_at: string;
  processed_at?: string;
  error_message?: string;
}
