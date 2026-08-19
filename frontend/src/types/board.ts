export type BoardKind = "lines" | "diagram" | "code" | "3d" | "clear";

export interface Scene3DPart {
  label: string;
  shape: "sphere" | "box" | "cylinder" | "cone" | "torus" | "capsule";
  color?: string;
  size?: number;
  x: number;
  y: number;
  z: number;
}

export interface Scene3DLink {
  from: string;
  to: string;
  label?: string;
}

export interface Scene3D {
  caption?: string;
  // Set for a real, curated anatomy model (see lib/board/anatomyAssets.ts) -
  // parts/links are meaningless when this is set. Empty for a procedural
  // scene instead.
  asset_key?: string;
  parts?: Scene3DPart[];
  links?: Scene3DLink[];
}

export interface BoardContentBlock {
  kind: BoardKind;
  title?: string;
  lines?: string[];
  mermaid?: string;
  code?: string;
  code_language?: string;
  scene3d?: Scene3D;
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
  // Set only for a session started from a learning-plan step (see
  // features/learning-plans/lib/api.ts's startPlanStep) - used to build
  // "continue learning" history on the learning-plans page.
  learning_plan_id?: string;
  learning_plan_step_index?: number;
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
